/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
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
		// TODO: manage password and read file
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

func (s *SshCExecutor) sshInteractive(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
	answers = make([]string, len(questions))
	// The second parameter is unused
	for n, _ := range questions {
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
			ssh.KeyboardInteractive(s.sshInteractive),
		}
	} else {

		if s.PrivateKeyPass != "" {
			return fmt.Errorf("Ssh private key with password not yet implemented")
		}

		signer, err := ssh.ParsePrivateKey([]byte(s.PrivateKey))
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
	if s.SftpClient == nil {
		client, err := sftp.NewClient(s.Client, opts...)
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
