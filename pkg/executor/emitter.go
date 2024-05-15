/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"io"
	//"fmt"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

type SshCEmitter struct {
	HostWriterStdout io.WriteCloser
	HostWriterStderr io.WriteCloser
	SshWriterStdout  io.WriteCloser
	SshWriterStderr  io.WriteCloser
}

func NewSshCEmitter() *SshCEmitter {
	return &SshCEmitter{
		HostWriterStdout: NewSshCEmitterWriter("host_stdout"),
		HostWriterStderr: NewSshCEmitterWriter("host_stderr"),
		SshWriterStdout:  NewSshCEmitterWriter("ssh_stdout"),
		SshWriterStderr:  NewSshCEmitterWriter("ssh_stderr"),
	}
}

func (e *SshCEmitter) GetHostWriterStdout() io.WriteCloser  { return e.HostWriterStdout }
func (e *SshCEmitter) GetHostWriterStderr() io.WriteCloser  { return e.HostWriterStderr }
func (e *SshCEmitter) SetHostWriterStdout(w io.WriteCloser) { e.HostWriterStdout = w }
func (e *SshCEmitter) SetHostWriterStderr(w io.WriteCloser) { e.HostWriterStderr = w }

func (e *SshCEmitter) GetSshWriterStdout() io.WriteCloser  { return e.SshWriterStdout }
func (e *SshCEmitter) GetSshWriterStderr() io.WriteCloser  { return e.SshWriterStderr }
func (e *SshCEmitter) SetSshWriterStdout(w io.WriteCloser) { e.SshWriterStdout = w }
func (e *SshCEmitter) SetSshWriterStderr(w io.WriteCloser) { e.SshWriterStderr = w }

func (e *SshCEmitter) DebugLog(color bool, args ...interface{}) {
	log.GetDefaultLogger().Msg("debug", color, true, args...)
}

func (e *SshCEmitter) InfoLog(color bool, args ...interface{}) {
	log.GetDefaultLogger().Msg("info", color, true, args...)
}

func (e *SshCEmitter) WarnLog(color bool, args ...interface{}) {
	log.GetDefaultLogger().Msg("warning", color, true, args...)
}

func (e *SshCEmitter) ErrorLog(color bool, args ...interface{}) {
	log.GetDefaultLogger().Msg("error", color, true, args...)
}

func (e *SshCEmitter) Emits(eType SshCExecutorEvent, data map[string]interface{}) {
	//logger := log.GetDefaultLogger()

	// TODO: review management of the setup event. We reload config too many times.
	//switch eType {
	//}
}
