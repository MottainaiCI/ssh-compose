/*
Copyright © 2024-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package core

import (
	"bufio"
	"io"

	"golang.org/x/crypto/ssh"
)

type CiscoCommandOpts struct {
	WithEna              bool
	OverrideDeadlineSecs int
}

type SshCSession struct {
	*ssh.Session
	Name string

	// Pipes (using C as prefix to avoid conflict with ssh.Session functions)
	CstdinPipe     io.WriteCloser
	CstdoutPipe    io.Reader
	CstderrPipe    io.Reader
	CstderrPipeBuf *bufio.Reader
	CstdoutPipeBuf *bufio.Reader

	CiscoPrompt    string
	CiscoEnaPrompt string
	InEna          bool
}

func NewCiscoCommandOpts(withEna bool) *CiscoCommandOpts {
	return &CiscoCommandOpts{
		WithEna:              withEna,
		OverrideDeadlineSecs: 3,
	}
}

func NewSshCSession(name string, s *ssh.Session) *SshCSession {
	return &SshCSession{
		Session: s,
		Name:    name,
	}
}

func (s *SshCSession) GetName() string             { return s.Name }
func (s *SshCSession) GetRawSession() *ssh.Session { return s.Session }
func (s *SshCSession) IsInCiscoEna() bool          { return s.InEna }
func (s *SshCSession) GetCiscoPrompt() string      { return s.CiscoPrompt }
func (s *SshCSession) GetCiscoEnaPrompt() string   { return s.CiscoEnaPrompt }

func (s *SshCSession) SetupPipes() {
	s.CstdinPipe, _ = s.StdinPipe()
	s.CstdoutPipe, _ = s.StdoutPipe()
	s.CstdoutPipeBuf = bufio.NewReader(s.CstdoutPipe)
}
