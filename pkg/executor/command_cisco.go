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
	"strconv"
	"strings"
	"time"

	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDevice(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string) (int, error) {
	return e.RunCommandWithOutputOnCiscoDeviceWithDS(
		nodeName, command, envs, outBuffer, errBuffer, entryPoint, 3)
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDeviceWithDS(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string, deadlineSecs int) (int, error) {

	if outBuffer == nil {
		return 1, errors.New("Invalid outBuffer")
	}
	if errBuffer == nil {
		return 1, errors.New("Invalid errBuffer")
	}

	termH := 200
	termW := 80
	dlSec := 3
	waitMs := 80

	var session *SshCSession
	var err error
	var present bool
	var output string
	firstLine := true
	buff := make([]byte, 80)
	logger := log.GetDefaultLogger()

	// Retrieve height and width from remote option
	height := e.GetOption("height")
	if height != "" {
		oh, _ := strconv.Atoi(height)
		if oh > 0 {
			termH = oh
		}
	}
	width := e.GetOption("width")
	if width != "" {
		ow, _ := strconv.Atoi(width)
		if ow > 0 {
			termW = ow
		}
	}
	// Retrieve deadline_secs from remote option
	dls := e.GetOption("deadline_secs")
	if dls != "" {
		odls, _ := strconv.Atoi(dls)
		if odls > 0 {
			dlSec = odls
		}
		if deadlineSecs > odls {
			dlSec = deadlineSecs
		}
	}
	// Retrieve wait_ms from remote option
	waitms := e.GetOption("wait_ms")
	if waitms != "" {
		owms, _ := strconv.Atoi(waitms)
		if owms > 0 {
			waitMs = owms
		}
	}

	// Always use the session with the name of the endpoint
	session, present = e.Sessions[e.Endpoint]
	if !present {

		term := os.Getenv("TERM")
		if term == "" {
			term = "linux"
			//term := "vt100"
		}
		// NOTE: Cisco devices old ignore this option. I don't see differences.
		//       The first line is always the command written.
		disableEchoShell := true

		logger.Debug(fmt.Sprintf("[%s] Using term size %d x %d with deadline secs %d, wait ms %d",
			e.Endpoint, termH, termW, dlSec, waitMs))

		session, err = e.GetShellSession(e.Endpoint, term, termH, termW, disableEchoShell)
		if err != nil {
			return 1, fmt.Errorf("on get session: %s", err.Error())
		}

		session.stdinPipe, _ = session.StdinPipe()
		session.stdoutPipe, _ = session.StdoutPipe()
		session.stdoutPipeBuf = bufio.NewReader(session.stdoutPipe)

		// It seems that the stderr is not used on Cisco Devices.

		// Initialize shell
		if err := session.Shell(); err != nil {
			return 1, fmt.Errorf("failed to start shell: %v", err)
		}

		// Get time to device to write the prompt. Maybe could be
		// set in the remote config option.
		time.Sleep(1000 * time.Millisecond)

		n, err := session.stdoutPipe.Read(buff)
		if err != nil {
			return 1, fmt.Errorf("failed on read prompt: %v", err)
		}

		// Ignore the first CR + LN send by device before the prompt.
		// This seems happens not always.
		if buff[0] == '\r' && buff[1] == '\n' {
			session.CiscoPrompt = string(buff[2:n])
		} else {
			session.CiscoPrompt = string(buff[0:n])
		}

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
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", dlSec))
	deadline := time.Now().Add(duration)
	waitMsDuration, _ := time.ParseDuration(fmt.Sprintf("%dms", waitMs))
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
		time.Sleep(waitMsDuration)
	}

	// Write the output in the buffer
	outBuffer.Write([]byte(output))

	e.Emitter.DebugLog(true,
		logger.Aurora.Bold(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] Command executed.", nodeName))))

	return 0, nil
}
