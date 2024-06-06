/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type SshCExecutor struct {
	Endpoint string
	Host     string
	// Ssh connection protocol. Valid values: tcp,tcp4,tcp6,unix
	ConnProtocol      string
	Port              int
	ShowCmdsOutput    bool
	RuntimeCmdsOutput bool
	Entrypoint        []string

	TTYOpISpeed uint32
	TTYOpOSpeed uint32

	User           string
	Pass           string
	PrivateKey     string
	PrivateKeyPass string

	Client     *ssh.Client
	SftpClient *sftp.Client

	Sessions map[string]*SshCSession

	Emitter SshCExecutorEmitter

	// ConfigDir of SSHC_CONF
	ConfigDir string
}

type SshCSession struct {
	*ssh.Session
	Name string
}

type TermShellRestoreCb func() error

func NewSshCSession(name string, s *ssh.Session) *SshCSession {
	return &SshCSession{
		Session: s,
		Name:    name,
	}
}

func (s *SshCSession) GetName() string             { return s.Name }
func (s *SshCSession) GetRawSession() *ssh.Session { return s.Session }

func NewSshCExecutor(endpoint, host string, port int) *SshCExecutor {
	return &SshCExecutor{
		Endpoint:          endpoint,
		Host:              host,
		Port:              port,
		ConnProtocol:      "tcp",
		ShowCmdsOutput:    true,
		RuntimeCmdsOutput: true,
		Entrypoint:        []string{},
		Client:            nil,
		SftpClient:        nil,
		Sessions:          make(map[string]*SshCSession, 0),
		Emitter:           NewSshCEmitter(),
		TTYOpOSpeed:       14400, // input speed = 14.4kbaud
		TTYOpISpeed:       14400, // output speed = 14.4kbaud
	}
}

func NewSshCExecutorFromRemote(rname string, r *specs.Remote) (*SshCExecutor, error) {
	ans := NewSshCExecutor(rname, r.Host, r.Port)
	ans.ConnProtocol = r.Protocol
	ans.User = r.User
	if r.AuthMethod == specs.AuthMethodPassword {
		ans.Pass = r.Pass
	} else {
		ans.PrivateKeyPass = r.PrivateKeyPass

		if r.PrivateKeyFile != "" {
			data, err := os.ReadFile(r.PrivateKeyFile)
			if err != nil {
				return ans, err
			}

			ans.PrivateKey = string(data)
		} else {
			ans.PrivateKey = r.PrivateKeyRaw
		}

	}
	return ans, nil
}

func (s *SshCExecutor) getSigner() (ssh.Signer, error) {
	var err error
	var signer ssh.Signer

	// Analyze key to check is valid and/or encrypted.
	pemblock, _ := pem.Decode([]byte(s.PrivateKey))
	if pemblock == nil {
		return nil, fmt.Errorf("Pem decode failed, no key found")
	}

	// Check if the key is encrypted.
	// NOTE: IsEncryptedPEMBlock and DecryptPEMBlock are deprecated
	//       in go. This code must be reviewed.
	if x509.IsEncryptedPEMBlock(pemblock) {
		if s.PrivateKeyPass == "" {
			return nil, fmt.Errorf("Found private key encrypted but no password defined.")
		}

		// decrypt PEM
		pemblock.Bytes, err = x509.DecryptPEMBlock(pemblock,
			[]byte(s.PrivateKeyPass))
		if err != nil {
			return nil, fmt.Errorf("error on decrypting PEM: %s", err.Error())
		}

		switch pemblock.Type {
		case "RSA PRIVATE KEY":
			key, err := x509.ParsePKCS1PrivateKey(pemblock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("Parsing PKCS private key failed %v", err)
			}
			// generate signer instance from key
			signer, err = ssh.NewSignerFromKey(key)

		case "EC PRIVATE KEY":
			key, err := x509.ParseECPrivateKey(pemblock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("Parsing EC private key failed %v", err)
			}

			// generate signer instance from key
			signer, err = ssh.NewSignerFromKey(key)
			/*
				convert key to PEM decoded:
				privkeyBytes, _ := x509.MarshalECPrivateKey(key)
				s.PrivateKey = string(pem.EncodeToMemory(
					&pem.Block{
						Type:  "EC PRIVATE KEY",
						Bytes: privkeyBytes,
					},
				))
			*/

		case "DSA PRIVATE KEY":
			key, err := ssh.ParseDSAPrivateKey(pemblock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("Parsing DSA private key failed %v", err)
			}
			signer, err = ssh.NewSignerFromKey(key)
		default:
			return nil, fmt.Errorf("Parsing private key failed, unsupported key type %q", pemblock.Type)
		}
	} else {
		// OPENSSH are not detected as PEM encrypted.
		if s.PrivateKeyPass == "" {
			signer, err = ssh.ParsePrivateKey([]byte(s.PrivateKey))
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(
				[]byte(s.PrivateKey),
				[]byte(s.PrivateKeyPass),
			)
		}

	}

	return signer, err
}

func (s *SshCExecutor) sshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	answers = make([]string, len(questions))
	// The second parameter is unused
	for n := range questions {
		//fmt.Println("Question ", v)
		answers[n] = s.Pass
	}

	return answers, nil
}

func (s *SshCExecutor) Close() {

	// Close all sessions
	if len(s.Sessions) > 0 {
		for name, session := range s.Sessions {
			session.Close()
			delete(s.Sessions, name)
		}
	}

	if s.SftpClient != nil {
		s.SftpClient.Close()
		s.SftpClient = nil
	}

	if s.Client != nil {
		s.Client.Close()
		s.Client = nil
	}
}

func (s *SshCExecutor) Setup() error {
	var err error

	conf := &ssh.ClientConfig{
		User:            s.User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // XXX: Security issue
	}

	if s.Pass != "" {
		conf.Auth = []ssh.AuthMethod{
			ssh.Password(s.Pass),
			ssh.KeyboardInteractive(s.sshInteractive),
		}
	} else {
		signer, err := s.getSigner()
		if err != nil {
			return err
		}

		conf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}
	}

	s.Client, err = ssh.Dial(s.ConnProtocol,
		strings.Join([]string{s.Host, ":", fmt.Sprintf("%d", s.Port)}, ""), conf)

	return err
}

func (s *SshCExecutor) SetupSftp(opts ...sftp.ClientOption) error {
	var client *sftp.Client
	var err error
	if s.SftpClient == nil {
		client, err = sftp.NewClient(s.Client, opts...)
		if err != nil {
			return err
		}
		s.SftpClient = client
	}
	return nil
}

func (e *SshCExecutor) GetEmitter() SshCExecutorEmitter        { return e.Emitter }
func (e *SshCExecutor) SetEmitter(emitter SshCExecutorEmitter) { e.Emitter = emitter }
func (s *SshCExecutor) GetClient() *ssh.Client                 { return s.Client }
func (s *SshCExecutor) GetSftpClient() *sftp.Client            { return s.SftpClient }
func (s *SshCExecutor) GetEndpoint() string                    { return s.Endpoint }
func (s *SshCExecutor) GetHost() string                        { return s.Host }
func (s *SshCExecutor) GetPort() int                           { return s.Port }
func (s *SshCExecutor) GetUser() string                        { return s.User }
func (s *SshCExecutor) GetPass() string                        { return s.Pass }
func (s *SshCExecutor) GetPrivateKey() string                  { return s.PrivateKey }
func (s *SshCExecutor) GetPrivateKeyPass() string              { return s.PrivateKeyPass }
func (s *SshCExecutor) GetConnProtocol() string                { return s.ConnProtocol }
func (s *SshCExecutor) GetShowCmdsOutput() bool                { return s.ShowCmdsOutput }
func (s *SshCExecutor) GetRuntimeCmdsOutput() bool             { return s.RuntimeCmdsOutput }

func (s *SshCExecutor) RemoveSession(n string) error {
	session, err := s.GetSession(n)
	if err != nil {
		return err
	}

	session.Close()
	delete(s.Sessions, n)

	return nil
}

func (s *SshCExecutor) ResetSession(n string) (*SshCSession, error) {
	session, err := s.GetSession(n)
	if err != nil {
		return nil, err
	}

	session.Close()
	delete(s.Sessions, n)

	sNew, err := s.GetSession(n)
	if err != nil {
		return nil, err
	}

	sNew.Stdout = session.Stdout
	sNew.Stdin = session.Stdin
	sNew.Stderr = session.Stderr

	return sNew, nil
}

func (s *SshCExecutor) GetSession(n string) (*SshCSession, error) {
	if _, ok := s.Sessions[n]; ok {
		return s.Sessions[n], nil
	}

	session, err := s.Client.NewSession()
	if err != nil {
		return nil, err
	}

	s.Sessions[n] = NewSshCSession(n, session)
	return s.Sessions[n], nil
}

func (s *SshCExecutor) GetShellSession(n, termType string, h, w int, echo bool) (*SshCSession, error) {
	session, err := s.GetSession(n)
	if err != nil {
		return nil, err
	}

	echoValue := uint32(0)
	if echo {
		echoValue = uint32(1)
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          echoValue, // disable echoing
		ssh.TTY_OP_OSPEED: s.TTYOpOSpeed,
		ssh.TTY_OP_ISPEED: s.TTYOpISpeed,
	}

	if err := session.RequestPty(termType, h, w, modes); err != nil {
		return nil, fmt.Errorf(
			"error on request pseudo terminal: %s", err.Error())
	}

	return session, nil
}

func (s *SshCExecutor) GetShellSessionWithTermSetup(n, termType string,
	stdin *os.File, stdout, stderr io.Writer) (*SshCSession, TermShellRestoreCb, error) {
	session, err := s.GetSession(n)
	if err != nil {
		return nil, nil, err
	}

	modes := ssh.TerminalModes{
		ssh.ECHO:          1, // disable echoing
		ssh.TTY_OP_OSPEED: s.TTYOpOSpeed,
		ssh.TTY_OP_ISPEED: s.TTYOpISpeed,
	}

	fd := int(stdin.Fd())

	w, h, err := terminal.GetSize(fd)
	if err != nil {
		return nil, nil, fmt.Errorf("error on retrieve terminal size: %s", err.Error())
	}

	state, err := terminal.MakeRaw(fd)
	if err != nil {
		return nil, nil, fmt.Errorf("error on terminal makeraw: %s", err.Error())
	}
	restoreCb := func() error {
		return terminal.Restore(fd, state)
	}

	if err := session.RequestPty(termType, h, w, modes); err != nil {
		return nil, nil, fmt.Errorf(
			"error on request pseudo terminal: %s", err.Error())
	}

	// set input and output
	session.Stdout = stdout
	session.Stdin = stdin
	session.Stderr = stderr

	return session, restoreCb, nil
}
