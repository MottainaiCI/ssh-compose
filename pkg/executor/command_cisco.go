/*
Copyright © 2024-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/MottainaiCI/ssh-compose/pkg/executor/cisco"
	"github.com/MottainaiCI/ssh-compose/pkg/executor/core"
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"
)

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDevice(nodeName, command string,
	envs map[string]string, outBuffer, errBuffer io.WriteCloser,
	entryPoint []string) (int, error) {

	return e.RunCommandWithOutputOnCiscoDeviceWithDS(
		nodeName, command, envs, outBuffer, errBuffer, entryPoint, core.NewCiscoCommandOpts(false))
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDeviceWithDS(nodeName, command string,
	envs map[string]string, outBuffer, errBuffer io.WriteCloser,
	entryPoint []string, opts *core.CiscoCommandOpts) (int, error) {

	if outBuffer == nil {
		return 1, errors.New("Invalid outBuffer")
	}
	if errBuffer == nil {
		return 1, errors.New("Invalid errBuffer")
	}

	var session *core.SshCSession
	var err error
	var present bool
	var output string
	logger := log.GetDefaultLogger()

	profileName := cisco.CiscoProfileTypeGeneral
	if e.GetOption(specs.OptionCiscoProfile) != "" {
		profileName = e.GetOption(specs.OptionCiscoProfile)
	}

	profile := cisco.NewCiscoProfile(profileName, e.CiscoPrompt,
		e.CiscoEnaPrompt, e.CiscoEnaPass, e.CiscoPromptRegex)

	task, _ := profile.GetTask(nodeName, &e.Options, opts)

	waitMsDuration, _ := time.ParseDuration(fmt.Sprintf("%dms", task.WaitMs))

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

		logger.Debug(fmt.Sprintf("[%s] Using term size %d x %d with deadline secs %d, wait ms %d. Profile %s.",
			e.Endpoint, task.TermHeight, task.TermWidth, task.DeadlineSec, task.WaitMs, profileName))

		session, err = e.GetShellSession(e.Endpoint, term, task.TermHeight, task.TermWidth, disableEchoShell)
		if err != nil {
			return 1, fmt.Errorf("on get session: %s", err.Error())
		}

		task.Session = session

		session.CstdinPipe, _ = session.StdinPipe()
		session.CstdoutPipe, _ = session.StdoutPipe()
		session.CstdoutPipeBuf = bufio.NewReader(session.CstdoutPipe)
		//session.SetupPipes()

		// It seems that the stderr is not used on Cisco Devices.

		// Initialize shell
		if err := session.Shell(); err != nil {
			return 1, fmt.Errorf("failed to start shell: %v", err)
		}

		output, err = profile.Login(nodeName, e.Emitter, task, session, opts)
		if err != nil {
			return 1, err
		}

	} else if e.CiscoPromptRegex {
		// Setup the regex object

		// Compile and validate Regex
		if e.CiscoPrompt != "" {
			task.CiscoPromptMatchRegex = regexp.MustCompile(e.CiscoPrompt)
			if task.CiscoPromptMatchRegex == nil {
				return 1, fmt.Errorf("failed on compile cisco prompt regex from value %s", e.CiscoPrompt)
			}
		}

		if e.CiscoEnaPrompt != "" {
			task.CiscoPromptEnaMatchRegex = regexp.MustCompile(e.CiscoEnaPrompt)
			if task.CiscoPromptEnaMatchRegex == nil {
				return 1, fmt.Errorf("failed on compile cisco ena prompt regex from value %s", e.CiscoEnaPrompt)
			}
		}
	}

	e.Emitter.InfoLog(true, logger.Aurora.Bold(
		logger.Aurora.Italic(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] - %s - :coffee:", nodeName, command)))))

	// Send command through stdin.
	// The number of \n (LF) depends of devices. For example
	// with ASA it's better two LF and on 3750 just one LF.
	suffix := "\r"
	for i := 0; i < task.NLF; i++ {
		suffix += "\n"
	}
	_, err = session.CstdinPipe.Write([]byte(command + suffix))

	if err != nil {
		return 1, fmt.Errorf("failed to write command %s: %s", command, err.Error())
	}

	task.FirstLine = true
	task.UsedPrompt = session.CiscoPrompt
	if session.InEna {
		task.UsedPrompt = session.CiscoEnaPrompt
	}
	duration, _ := time.ParseDuration(fmt.Sprintf("%ds", task.DeadlineSec))
	deadline := time.Now().Add(duration)

	if len(task.UsedPrompt) > 2 {
		// NOTE: Cisco ASA Firepower 2130 send a space in the initial prompt but
		//       not at the end of the execution of a command.
		//       So it's better check if HasPrefix match also without it
		task.UsedPromptHasFinalSpace = task.UsedPrompt[len(task.UsedPrompt)-1:len(task.UsedPrompt)] == " "
	}

	logger.Debug(fmt.Sprintf("[%s] Ena %v, PromptRegex %v, usedPromptHasFinalSpace %v. Using prompt '%s'",
		e.Endpoint, session.InEna, e.CiscoPromptRegex, task.UsedPromptHasFinalSpace, task.UsedPrompt))

	i := 1
	for {
		if time.Now().After(deadline) {
			break
		}

		if i > 2 && task.FirstLine {
			logger.Debug(fmt.Sprintf("[%s] First line never skipped."), e.Endpoint)
		}

		line, err := session.CstdoutPipeBuf.ReadString('\n')
		if err != nil && err != io.EOF {
			return 1, fmt.Errorf("read error: %w", err)
		}

		if task.FirstLine {
			// Skip first line with the written command.
			if profile.IsCommandEchoLine(command, line, task, session) {
				logger.Debug(fmt.Sprintf("[%s] Catch command line with %s",
					e.Endpoint, line))
				task.FirstLine = false
				continue
			}

		}

		output = output + line

		if profile.IsPromptLine(line, task, session) {
			logger.Debug(fmt.Sprintf("[%s] Catch prompt line with:\n%s",
				e.Endpoint, line))
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
