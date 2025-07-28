/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDevice(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string) (int, error) {

	if outBuffer == nil {
		return 1, errors.New("Invalid outBuffer")
	}
	if errBuffer == nil {
		return 1, errors.New("Invalid errBuffer")
	}

	var session *SshCSession
	var err error
	var present bool
	var output string
	firstLine := true
	buff := make([]byte, 80)
	logger := log.GetDefaultLogger()

	// Always use the session with the name of the endpoint
	session, present = e.Sessions[e.Endpoint]
	if !present {

		term := os.Getenv("TERM")
		if term == "" {
			term = "linux"
			//term := "vt100"
		}
		disableEchoShell := true
		session, err = e.GetShellSession(e.Endpoint, term, 80, 40, disableEchoShell)
		if err != nil {
			return 1, fmt.Errorf("on get session: %s", err.Error())
		}

		session.stdinPipe, _ = session.StdinPipe()
		session.stdoutPipe, _ = session.StdoutPipe()
		session.stdoutPipeBuf = bufio.NewReader(session.stdoutPipe)

		// We ignore stderr for now.
		//stderr, _ := session.StderrPipe()
		//session.stderrPipe = bufio.NewReader(stderr)

		// Initialize shell
		if err := session.Shell(); err != nil {
			return 1, fmt.Errorf("failed to start shell: %v", err)
		}

		// Get time to device to write the prompt. Maybe could be
		// set in the remote config option.
		time.Sleep(400 * time.Millisecond)

		n, err := session.stdoutPipe.Read(buff)
		if err != nil {
			return 1, fmt.Errorf("failed on read prompt: %v", err)
		}

		// Ignore the first CR + LN send by device before the prompt
		session.CiscoPrompt = string(buff[2:n])

		if e.CiscoPrompt != "" && session.CiscoPrompt != e.CiscoPrompt {
			e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt %s (session) != %s (config)",
				e.Endpoint, session.CiscoPrompt, e.CiscoPrompt))
		} else if e.CiscoPrompt == "" {
			e.Emitter.InfoLog(true, logger.Aurora.Bold(
				logger.Aurora.Italic(
					logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] - Using cisco prompt %s - :eye:", nodeName, session.CiscoPrompt)))))
		}
	}

	e.Emitter.InfoLog(true, logger.Aurora.Bold(
		logger.Aurora.Italic(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] - %s - :coffee:", nodeName, command)))))

	// Send command through stdin
	_, err = session.stdinPipe.Write([]byte(command + "\r\n"))
	if err != nil {
		return 1, fmt.Errorf("failed to write command %s: %s", command, err.Error())
	}

	firstLine = true
	deadline := time.Now().Add(3 * time.Second)
	for {
		if time.Now().After(deadline) {
			break
		}

		line, err := session.stdoutPipeBuf.ReadString('\n')
		if err != nil && err != io.EOF {
			return 1, fmt.Errorf("read error: %w", err)
		}

		if firstLine {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, command) || strings.HasPrefix(line, session.CiscoPrompt+command) {
				firstLine = false
				continue
			}
		}

		output += line

		if strings.HasPrefix(line, session.CiscoPrompt) {
			break
		}

		// Waiting a bit
		time.Sleep(100 * time.Millisecond)
	}

	// Write the output in the buffer
	outBuffer.Write([]byte(output))

	e.Emitter.DebugLog(true,
		logger.Aurora.Bold(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] Command executed.", nodeName))))

	return 0, nil
}
