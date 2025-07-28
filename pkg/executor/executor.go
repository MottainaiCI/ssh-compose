/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"bufio"
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type TunnelHop struct {
	// Ssh connection protocol. Valid values: tcp,tcp4,tcp6,unix
	ConnProtocol string
	Host         string
	Port         int
	TimeoutSecs  *uint

	User           string
	Pass           string
	PrivateKey     string
	PrivateKeyPass string

	Client *ssh.Client
}

type SshCExecutor struct {
	Endpoint string
	Host     string
	// Ssh connection protocol. Valid values: tcp,tcp4,tcp6,unix
	ConnProtocol      string
	Port              int
	TimeoutSecs       *uint
	ShowCmdsOutput    bool
	RuntimeCmdsOutput bool
	Entrypoint        []string

	// Cisco Device options
	CiscoDevice bool
	CiscoPrompt string

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

	// Context used to manage all SSL sessions
	Ctx    context.Context
	Cancel context.CancelFunc

	TunnelChain     []*TunnelHop
	TunnelLocalPort int
	TunnelLocalAddr string
	TunnelLocalBind bool
	LocalListener   net.Listener
	LocalListenerWg sync.WaitGroup
}

type SshCSession struct {
	*ssh.Session
	Name string

	// Pipes
	stdinPipe     io.WriteCloser
	stdoutPipe    io.Reader
	stderrPipeBuf *bufio.Reader
	stdoutPipeBuf *bufio.Reader

	CiscoPrompt string
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

func NewTunnelHop(r *specs.Remote) (*TunnelHop, error) {
	ans := &TunnelHop{
		ConnProtocol: r.Protocol,
		User:         r.User,
		Host:         r.Host,
		Port:         r.Port,
		TimeoutSecs:  r.TimeoutSecs,
	}

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
		TunnelLocalAddr:   "localhost",
	}
}

func NewSshCExecutorFromRemote(rname string, r *specs.Remote) (*SshCExecutor, error) {
	ans := NewSshCExecutor(rname, r.Host, r.Port)
	ans.ConnProtocol = r.Protocol
	ans.User = r.User
	ans.CiscoDevice = r.CiscoDevice
	ans.CiscoPrompt = r.CiscoPrompt
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

	ans.TunnelLocalPort = r.TunLocalPort
	ans.TunnelLocalAddr = r.TunLocalAddr
	ans.TunnelLocalBind = r.TunLocalBind
	ans.TimeoutSecs = r.TimeoutSecs

	if r.HasChain() {
		for _, cr := range r.GetChain() {
			tun, err := NewTunnelHop(&cr)
			if err != nil {
				return ans, err
			}
			ans.TunnelChain = append(ans.TunnelChain, tun)
		}
	}

	return ans, nil
}

func (s *SshCExecutor) getSigner(privateKey, privateKeyPass string) (ssh.Signer, error) {
	var err error
	var signer ssh.Signer

	// Analyze key to check is valid and/or encrypted.
	pemblock, _ := pem.Decode([]byte(privateKey))
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
		if privateKeyPass == "" {
			signer, err = ssh.ParsePrivateKey([]byte(privateKey))
		} else {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(
				[]byte(privateKey),
				[]byte(privateKeyPass),
			)
		}

	}

	return signer, err
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

	if len(s.TunnelChain) > 0 {

		// Call context cancel
		if s.Cancel != nil {
			s.Cancel()
		}

		// Close tunnels session in the reverse order
		for i := len(s.TunnelChain) - 1; i > 0; i-- {
			if s.TunnelChain[i].Client != nil {
				s.TunnelChain[i].Client.Close()
			}
		}

		if s.TunnelLocalBind {
			s.LocalListener.Close()

			s.LocalListenerWg.Wait()
		}
	}

}

func (s *SshCExecutor) BuildChain() (*ssh.Client, error) {
	logger := log.GetDefaultLogger()

	var client *ssh.Client
	var err error

	for idx := range s.TunnelChain {
		conf, err := s.getSshClientConfig(
			s.TunnelChain[idx].User,
			s.TunnelChain[idx].Pass,
			s.TunnelChain[idx].PrivateKey,
			s.TunnelChain[idx].PrivateKeyPass,
			s.TunnelChain[idx].TimeoutSecs)
		if err != nil {
			return nil, err
		}

		if idx == 0 {
			// POST: First hop, creating direct dial connection.

			logger.DebugC(fmt.Sprintf(
				"[%s] Connecting to first hop at %s:%d...",
				s.Endpoint, s.TunnelChain[idx].Host, s.TunnelChain[idx].Port))

			client, err = ssh.Dial(s.TunnelChain[idx].ConnProtocol,
				strings.Join([]string{
					s.TunnelChain[idx].Host, ":",
					fmt.Sprintf("%d", s.TunnelChain[idx].Port)}, ""), conf)
			if err != nil {
				return nil, err
			}
		} else {
			// POST: hop >=2 -> we need to using the previous hop ssh client.

			logger.DebugC(fmt.Sprintf(
				"[%s] Connecting to hop %d at %s:%d...",
				s.Endpoint, idx+1, s.TunnelChain[idx].Host, s.TunnelChain[idx].Port))

			targetAddr := fmt.Sprintf("%s:%d", s.TunnelChain[idx].Host,
				s.TunnelChain[idx].Port)

			// Create connection dialer (through the previous hop client)
			conn, err := client.Dial(s.TunnelChain[idx].ConnProtocol, targetAddr)
			if err != nil {
				return nil, err
			}

			connCh, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, conf)
			if err != nil {
				return nil, err
			}

			client = ssh.NewClient(connCh, chans, reqs)
		}

		s.TunnelChain[idx].Client = client
	}

	if s.TunnelLocalBind {
		// Creating local binding port in order to easily use
		// the tunnel from external tools.

		// Create the local listening port binding for
		// final ssh connection.
		listenAddr := fmt.Sprintf("%s:%d", s.TunnelLocalAddr, s.TunnelLocalPort)
		logger.DebugC(fmt.Sprintf(
			"[%s] Binding local tunnel at %s...", s.Endpoint, listenAddr))

		s.LocalListener, err = net.Listen("tcp", listenAddr)
		if err != nil {
			return client, err
		}

		// Client connection callback
		handleClientCb := func(ctx context.Context, localConn net.Conn,
			sshClient *ssh.Client, target, protocol string, wg *sync.WaitGroup) {

			defer wg.Done()
			defer localConn.Close()

			remoteConn, err := sshClient.Dial(protocol, target)
			if err != nil {
				logger.Warning(fmt.Sprintf(
					"[%s] failed to dial target %s for incoming connection %s: %s",
					s.Endpoint, target, localConn.RemoteAddr().String(),
					err.Error()))
				return
			}
			defer remoteConn.Close()

			done := make(chan struct{})

			// Crete goroutine to manage outcoming data
			go func() {
				io.Copy(remoteConn, localConn)
				if c, ok := remoteConn.(interface{ CloseWrite() error }); ok {
					c.CloseWrite()
				} else {
					remoteConn.Close()
				}
				done <- struct{}{}
			}()

			// Create goroutine to manage incoming data
			go func() {
				io.Copy(localConn, remoteConn)
				localConn.Close()
				done <- struct{}{}
			}()

			select {
			case <-ctx.Done():
			case <-done:
				<-done
			}

		}

		acceptCb := func() {
			targetAddr := fmt.Sprintf("%s:%d", s.Host, s.Port)
			for {
				conn, err := s.LocalListener.Accept()
				if err != nil {
					select {
					case <-s.Ctx.Done():
						logger.DebugC(fmt.Sprintf("[%s] listener closed, shutting down.",
							s.Endpoint))
						return
					default:
						logger.DebugC(fmt.Sprintf("[%s] accept error from %s: %s",
							s.Endpoint, conn.RemoteAddr().String(), err.Error()))
						continue
					}
				}
				s.LocalListenerWg.Add(1)
				go handleClientCb(s.Ctx, conn, client, targetAddr, s.ConnProtocol, &s.LocalListenerWg)
			}
		}

		go acceptCb()
	}

	return client, nil
}

func (s *SshCExecutor) getSshClientConfig(user, pass,
	privateKey, privateKeyPass string, timeout *uint) (*ssh.ClientConfig, error) {
	conf := &ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // XXX: Security issue
	}

	if pass != "" {
		conf.Auth = []ssh.AuthMethod{
			ssh.Password(pass),
			ssh.KeyboardInteractive(
				func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
					answers = make([]string, len(questions))
					// The second parameter is unused
					for n := range questions {
						answers[n] = pass
					}
					return answers, nil
				}),
		}
	} else {
		signer, err := s.getSigner(privateKey, privateKeyPass)
		if err != nil {
			return nil, err
		}

		conf.Auth = []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		}

	}

	if timeout != nil {
		duration, err := time.ParseDuration(fmt.Sprintf("%ds", timeout))
		if err == nil {
			conf.Timeout = duration
		}
	}

	return conf, nil
}

func (s *SshCExecutor) Setup() error {
	var err error
	var targetAddr string
	logger := log.GetDefaultLogger()

	if s.Client != nil {
		// Nothing to do.
		return nil
	}

	// Create the context used to manage
	// all SSL sessions.
	s.Ctx, s.Cancel = context.WithCancel(context.Background())

	conf, err := s.getSshClientConfig(s.User, s.Pass, s.PrivateKey, s.PrivateKeyPass,
		s.TimeoutSecs)
	if err != nil {
		return err
	}

	if len(s.TunnelChain) > 0 {
		tunClient, err := s.BuildChain()
		if err != nil {
			return err
		}

		if s.TunnelLocalBind {

			targetAddr = fmt.Sprintf("%s", s.LocalListener.Addr().String())
			logger.DebugC(fmt.Sprintf(
				"[%s] Connecting at %s to reach %s:%d...",
				s.Endpoint, targetAddr, s.Host, s.Port))

			s.Client, err = ssh.Dial(s.ConnProtocol, targetAddr, conf)
			if err != nil {
				return err
			}

		} else {

			targetAddr = fmt.Sprintf("%s:%d", s.Host, s.Port)

			logger.DebugC(fmt.Sprintf(
				"[%s] Connecting to %s ...", s.Endpoint, targetAddr))

			// Create connection dialer (through the previous hop client)
			conn, err := tunClient.Dial(s.ConnProtocol, targetAddr)
			if err != nil {
				return err
			}

			connCh, chans, reqs, err := ssh.NewClientConn(conn, targetAddr, conf)
			if err != nil {
				return err
			}

			s.Client = ssh.NewClient(connCh, chans, reqs)
		}

	} else {
		targetAddr = fmt.Sprintf("%s:%d", s.Host, s.Port)

		logger.DebugC(fmt.Sprintf(
			"[%s] Connecting to %s ...", s.Endpoint, targetAddr))

		s.Client, err = ssh.Dial(s.ConnProtocol, targetAddr, conf)

	}

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

	if s.Client == nil {
		return nil, fmt.Errorf("SSH Client not initialized")
	}

	if len(s.Sessions) > 0 && s.CiscoDevice {
		return nil, fmt.Errorf("Cisco device supports only one session")
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

	if err = session.RequestPty(termType, h, w, modes); err != nil {
		return nil, nil, fmt.Errorf(
			"error on request pseudo terminal: %s", err.Error())
	}

	// set input and output
	session.Stdout = stdout
	session.Stdin = stdin
	session.Stderr = stderr

	return session, restoreCb, nil
}
