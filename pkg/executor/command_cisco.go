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
	"github.com/MottainaiCI/ssh-compose/pkg/specs"
)

type CiscoCommandOpts struct {
	WithEna              bool
	OverrideDeadlineSecs int
}

func NewCiscoCommandOpts(withEna bool) *CiscoCommandOpts {
	return &CiscoCommandOpts{
		WithEna:              withEna,
		OverrideDeadlineSecs: 3,
	}
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDevice(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string) (int, error) {
	return e.RunCommandWithOutputOnCiscoDeviceWithDS(
		nodeName, command, envs, outBuffer, errBuffer, entryPoint, NewCiscoCommandOpts(false))
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDeviceWithDS(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string, opts *CiscoCommandOpts) (int, error) {

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
	bannerLines := 0

	var session *SshCSession
	var err error
	var present bool
	var output string
	firstLine := true
	buff := make([]byte, 80)
	logger := log.GetDefaultLogger()

	// Retrieve height and width from remote option
	height := e.GetOption(specs.OptionTermHeight)
	if height != "" {
		oh, _ := strconv.Atoi(height)
		if oh > 0 {
			termH = oh
		}
	}
	width := e.GetOption(specs.OptionTermWidth)
	if width != "" {
		ow, _ := strconv.Atoi(width)
		if ow > 0 {
			termW = ow
		}
	}
	// Retrieve deadline_secs from remote option
	dls := e.GetOption(specs.OptionDeadlineSecs)
	if dls != "" {
		odls, _ := strconv.Atoi(dls)
		if odls > 0 {
			dlSec = odls
		}
		if opts.OverrideDeadlineSecs > odls {
			dlSec = opts.OverrideDeadlineSecs
		}
	}
	// Retrieve wait_ms from remote option
	waitms := e.GetOption(specs.OptionWaitMs)
	if waitms != "" {
		owms, _ := strconv.Atoi(waitms)
		if owms > 0 {
			waitMs = owms
		}
	}
	// Retrieve banner lines option
	if e.GetOption(specs.OptionBannerLines) != "" {
		blines, _ := strconv.Atoi(e.GetOption(specs.OptionBannerLines))
		if blines > 0 {
			bannerLines = blines
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

		if bannerLines > 0 {

			banner := ""

			for i := 0; i < bannerLines; i++ {

				line, err := session.stdoutPipeBuf.ReadString('\n')
				if err != nil {
					return 1, fmt.Errorf("read error: %w", err)
				}

				banner += line
			}

			logger.Debug(fmt.Sprintf("[%s] Skipped banner:\n%s---",
				e.Endpoint, banner))

			output += banner
		}

		n, err := session.stdoutPipeBuf.Read(buff)
		if err != nil {
			return 1, fmt.Errorf("failed on read prompt: %v", err)
		}

		// Ignore the first CR + LN send by device before the prompt
		// or only CR after the banner.
		// This seems happens not always.
		if buff[0] == '\r' && buff[1] == '\n' {
			session.CiscoPrompt = string(buff[2:n])
		} else if buff[0] == '\r' {
			session.CiscoPrompt = string(buff[1:n])
		} else {
			session.CiscoPrompt = string(buff[0:n])
		}

		if e.CiscoPrompt != "" && session.CiscoPrompt != e.CiscoPrompt {
			e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) != '%s' (config)",
				e.Endpoint, session.CiscoPrompt, e.CiscoPrompt))
		} else if e.CiscoPrompt == "" {
			e.Emitter.InfoLog(true, logger.Aurora.Bold(
				logger.Aurora.Italic(
					logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] - Using cisco prompt %s - :eye:", nodeName, session.CiscoPrompt)))))
		}

	}

	if opts.WithEna {
		// POST: The command requires ena privileges

		// Send ena command on stdin
		_, err = session.stdinPipe.Write([]byte("ena" + "\r\n"))
		if err != nil {
			return 1, fmt.Errorf("failed to write ena command: %w", err)
		}

		// Read the sent command
		line, _ := session.stdoutPipeBuf.ReadString('\n')

		output += line
		// Read the ask password output
		n, _ := session.stdoutPipeBuf.Read(buff)
		line = string(buff[0:n])

		output += line

		if !strings.Contains(line, "Password:") {
			return 1, fmt.Errorf("received invalid response for ena command: %s", line)
		}

		_, _ = session.stdinPipe.Write([]byte(e.CiscoEnaPass + "\r"))

		// Ignoring Response **** \r\n
		n, _ = session.stdoutPipeBuf.Read(buff)
		output += string(buff[0:n])
		// If i the password is not defined we send only \r\n and we need
		// to parse Invalid Password. If the password is correct then
		// in the buffer will arrive *****
		if n > 2 && buff[0] == '\r' && buff[1] == '\n' {
			return 1, fmt.Errorf("unexpected state on manage ena (%s)", string(buff[0:n]))
		}

		n, _ = session.stdoutPipeBuf.Read(buff)
		output += string(buff[0:n])
		if buff[0] == '\r' {
			line = string(buff[1:n])
		} else {
			line = string(buff[0:n])
		}
		if strings.Contains(line, "Invalid password") {
			return 1, fmt.Errorf("invalid ena credential")
		}

		// POST: if all works fine the line will contains the new prompt

		session.CiscoEnaPrompt = line
		if e.CiscoEnaPrompt != "" && session.CiscoEnaPrompt != e.CiscoEnaPrompt {
			e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on ena prompt '%s' (session) != '%s' (config)",
				e.Endpoint, session.CiscoEnaPrompt, e.CiscoEnaPrompt))
		} else if e.CiscoEnaPrompt == "" {
			e.Emitter.InfoLog(true, logger.Aurora.Bold(
				logger.Aurora.Italic(
					logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] - Using cisco ena prompt %s - :eye:", nodeName, session.CiscoEnaPrompt)))))
		}

		session.InEna = true
	}

	e.Emitter.InfoLog(true, logger.Aurora.Bold(
		logger.Aurora.Italic(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] - %s - :coffee:", nodeName, command)))))

	// Send command through stdin
	// I use two \n to force a new line in the prompt. We need to investigate in the
	// different devices. This works with Cisco ASA 5565 and Cisco 3750
	_, err = session.stdinPipe.Write([]byte(command + "\r\n\n"))
	if err != nil {
		return 1, fmt.Errorf("failed to write command %s: %s", command, err.Error())
	}

	firstLine = true
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", dlSec))
	deadline := time.Now().Add(duration)
	usedPrompt := session.CiscoPrompt
	if session.InEna {
		usedPrompt = session.CiscoEnaPrompt
	}

	logger.Debug(fmt.Sprintf("[%s] Ena %v. Using prompt '%s'",
		e.Endpoint, session.InEna, usedPrompt))

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
			// Skip first line with the written command.
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, command) || strings.HasPrefix(line, usedPrompt+command) {
				firstLine = false
				continue
			}
		}

		output += line

		if strings.HasPrefix(line, usedPrompt) {
			break
		}

		// Check if arrive the line of the prompt with the first
		// char equals to \r. It seems that on ASA devices
		// at the end of the command output the device
		// send an additional \r
		if strings.Contains(line, usedPrompt) {
			if line[0] == '\r' {
				line = line[1:]
			}
			if strings.HasPrefix(line, usedPrompt) {
				break
			}
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
