/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"fmt"
	"os"
	"syscall"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"

	"golang.org/x/crypto/ssh/terminal"
)

func ResizeWindowHandler(sigs chan os.Signal, stdin *os.File, session *SshCSession) {
	logger := log.GetDefaultLogger()
	for true {
		sig := <-sigs
		switch sig {
		case syscall.SIGWINCH:
			logger.Debug(
				fmt.Sprintf("Received signal '%s', updating window geometry", sig))
			fd := int(stdin.Fd())
			if w, h, err := terminal.GetSize(fd); err == nil {
				_ = session.Session.WindowChange(h, w)
			}
		default:
			logger.Debug(fmt.Sprintf("Received signal '%s'. Exiting", sig))
			return
		}
	}
}
