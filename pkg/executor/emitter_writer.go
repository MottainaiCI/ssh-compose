/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

type SshCEmitterWriter struct {
	Type string
}

func NewSshCEmitterWriter(t string) *SshCEmitterWriter {
	return &SshCEmitterWriter{Type: t}
}

func (e *SshCEmitterWriter) Write(p []byte) (int, error) {
	logger := log.GetDefaultLogger()
	switch e.Type {
	case "ssh_stdout":
		logger.Msg("info", true, false, string(p))
	case "host_stdout":
		logger.Msg("info", false, false,
			logger.Aurora.Bold(
				logger.Aurora.BrightYellow(string(p)),
			),
		)
	case "host_stderr", "ssh_stderr":
		logger.Msg("info", false, false,
			logger.Aurora.BrightRed(string(p)))
	}
	return len(p), nil
}

func (e *SshCEmitterWriter) Close() error {
	return nil
}
