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

type CiscoExecTask struct {
	CiscoPromptMatchRegex    *regexp.Regexp
	CiscoPromptEnaMatchRegex *regexp.Regexp
	FirstLine                bool
	Session                  *SshCSession
	UsedPrompt               string
	UsedPromptHasFinalSpace  bool

	TermHeight  int
	TermWidth   int
	DeadlineSec int
	WaitMs      int
	NLF         int
	BannerLines int
	BannerShow  bool
}

func NewCiscoCommandOpts(withEna bool) *CiscoCommandOpts {
	return &CiscoCommandOpts{
		WithEna:              withEna,
		OverrideDeadlineSecs: 3,
	}
}

func (e *SshCExecutor) ciscoValidatePrompt(nodeName string, session *SshCSession, task *CiscoExecTask) error {
	logger := log.GetDefaultLogger()

	if e.CiscoPrompt != "" && (!e.CiscoPromptRegex) && session.CiscoPrompt != e.CiscoPrompt {
		// POST: CiscoPrompt is defined but is not a regex.
		e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) != '%s' (config)",
			e.Endpoint, session.CiscoPrompt, e.CiscoPrompt))
	} else if e.CiscoPromptRegex {
		// POST: CiscoPromptRegex is enable.

		if e.CiscoPrompt == "" {
			return fmt.Errorf("invalid configuration with empty prompt string and regex enabled")
		}

		// Compile and validate Regex
		task.CiscoPromptMatchRegex = regexp.MustCompile(e.CiscoPrompt)
		if task.CiscoPromptMatchRegex == nil {
			return fmt.Errorf("failed on compile cisco prompt regex from value %s", e.CiscoPrompt)
		}

		if task.CiscoPromptMatchRegex.MatchString(session.CiscoPrompt) {
			logger.Debug(fmt.Sprintf("[%s] %s regex match %s", e.Endpoint, e.CiscoPrompt, session.CiscoPrompt))
		} else {
			e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) for regex '%s' (config)",
				e.Endpoint, session.CiscoPrompt, e.CiscoPrompt))
		}

	} else if e.CiscoPrompt == "" {
		e.Emitter.InfoLog(true, logger.Aurora.Bold(
			logger.Aurora.Italic(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - Using cisco prompt %s - :eye:", nodeName, session.CiscoPrompt)))))
	}

	return nil
}

func (e *SshCExecutor) ciscoValidateEnaPrompt(nodeName string, session *SshCSession, task *CiscoExecTask) error {
	logger := log.GetDefaultLogger()

	if e.CiscoEnaPrompt != "" && (!e.CiscoPromptRegex) && session.CiscoEnaPrompt != e.CiscoEnaPrompt {
		e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on ena prompt '%s' (session) != '%s' (config)",
			e.Endpoint, session.CiscoEnaPrompt, e.CiscoEnaPrompt))

	} else if e.CiscoPromptRegex {
		// POST: CiscoPromptRegex is enable.

		if e.CiscoEnaPrompt == "" {
			return fmt.Errorf("invalid configuration with empty ena prompt string and regex enabled")
		}

		// Compile and validate Regex
		task.CiscoPromptEnaMatchRegex = regexp.MustCompile(e.CiscoEnaPrompt)
		if task.CiscoPromptEnaMatchRegex == nil {
			return fmt.Errorf("failed on compile cisco ena prompt regex from value %s", e.CiscoEnaPrompt)
		}

		if task.CiscoPromptEnaMatchRegex.MatchString(session.CiscoEnaPrompt) {
			logger.Debug(fmt.Sprintf("[%s] %s regex match %s", e.Endpoint, e.CiscoEnaPrompt, session.CiscoEnaPrompt))
		} else {
			e.Emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) for regex '%s' (config)",
				e.Endpoint, session.CiscoEnaPrompt, e.CiscoEnaPrompt))
		}

	} else if e.CiscoEnaPrompt == "" {
		e.Emitter.InfoLog(true, logger.Aurora.Bold(
			logger.Aurora.Italic(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - Using cisco ena prompt %s - :eye:", nodeName, session.CiscoEnaPrompt)))))
	}

	return nil
}

func (e *SshCExecutor) ciscoIsCommandEchoLine(command, line string,
	task *CiscoExecTask, session *SshCSession) bool {

	ans := false

	lineTrim := strings.TrimSpace(line)
	if strings.HasPrefix(lineTrim, command) {
		ans = true
	} else if e.CiscoPromptRegex {

		// Check if the line match the prompt regex and it contains the command
		if session.InEna {
			if task.CiscoPromptEnaMatchRegex.MatchString(lineTrim) && strings.Contains(lineTrim, command) {
				ans = true
			}
		} else {
			if task.CiscoPromptMatchRegex.MatchString(lineTrim) && strings.Contains(lineTrim, command) {
				ans = true
			}
		}

	} else if strings.HasPrefix(lineTrim, task.UsedPrompt+command) ||
		(task.UsedPromptHasFinalSpace && strings.HasPrefix(lineTrim, task.UsedPrompt[0:len(task.UsedPrompt)-1]+command)) {
		ans = true
	}

	return ans
}

func (e *SshCExecutor) ciscoIsPromptLine(line string, task *CiscoExecTask, session *SshCSession) bool {
	ans := false

	// Check if arrive the line of the prompt with the first
	// char equals to \r. It seems that on ASA devices
	// at the end of the command output the device
	// send an additional \r
	if len(line) > 2 && line[0] == '\r' {
		line = line[1:]
	}

	// Drop final CR + LN
	if len(line) > 2 && line[len(line)-1:] == "\n" {
		line = line[0 : len(line)-1]
	}
	if len(line) > 2 && line[len(line)-1:] == "\r" {
		line = line[0 : len(line)-1]
	}

	if e.CiscoPromptRegex {

		if session.InEna {

			if task.CiscoPromptEnaMatchRegex.MatchString(line) {
				ans = true
			}

		} else {

			if task.CiscoPromptMatchRegex.MatchString(line) {
				ans = true
			}

		}

	} else if strings.HasPrefix(line, task.UsedPrompt) ||
		(task.UsedPromptHasFinalSpace && strings.HasPrefix(line, task.UsedPrompt[0:len(task.UsedPrompt)-1])) {
		ans = true
	}

	return ans
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDevice(nodeName, command string,
	envs map[string]string, outBuffer, errBuffer io.WriteCloser,
	entryPoint []string) (int, error) {

	return e.RunCommandWithOutputOnCiscoDeviceWithDS(
		nodeName, command, envs, outBuffer, errBuffer, entryPoint, NewCiscoCommandOpts(false))
}

func (e *SshCExecutor) RunCommandWithOutputOnCiscoDeviceWithDS(nodeName, command string,
	envs map[string]string, outBuffer, errBuffer io.WriteCloser,
	entryPoint []string, opts *CiscoCommandOpts) (int, error) {

	if outBuffer == nil {
		return 1, errors.New("Invalid outBuffer")
	}
	if errBuffer == nil {
		return 1, errors.New("Invalid errBuffer")
	}

	task := &CiscoExecTask{
		FirstLine:               true,
		TermHeight:              200,
		TermWidth:               80,
		DeadlineSec:             3,
		WaitMs:                  80,
		NLF:                     1,
		BannerLines:             0,
		BannerShow:              true,
		UsedPromptHasFinalSpace: false,
	}

	var session *SshCSession
	var err error
	var present bool
	var output string
	buff := make([]byte, 1024)
	logger := log.GetDefaultLogger()

	// Retrieve height and width from remote option
	height := e.GetOption(specs.OptionTermHeight)
	if height != "" {
		oh, _ := strconv.Atoi(height)
		if oh > 0 {
			task.TermHeight = oh
		}
	}
	width := e.GetOption(specs.OptionTermWidth)
	if width != "" {
		ow, _ := strconv.Atoi(width)
		if ow > 0 {
			task.TermWidth = ow
		}
	}
	// Retrieve deadline_secs from remote option
	dls := e.GetOption(specs.OptionDeadlineSecs)
	if dls != "" {
		odls, _ := strconv.Atoi(dls)
		if odls > 0 {
			task.DeadlineSec = odls
		}
		if opts.OverrideDeadlineSecs > odls {
			task.DeadlineSec = opts.OverrideDeadlineSecs
		}
	}
	// Retrieve wait_ms from remote option
	waitms := e.GetOption(specs.OptionWaitMs)
	if waitms != "" {
		owms, _ := strconv.Atoi(waitms)
		if owms > 0 {
			task.WaitMs = owms
		}
	}
	// Retrieve banner lines option
	if e.GetOption(specs.OptionBannerLines) != "" {
		blines, _ := strconv.Atoi(e.GetOption(specs.OptionBannerLines))
		if blines > 0 {
			task.BannerLines = blines
		}
	}

	// Retrieve number of LF (line feed) to append in the command.
	if e.GetOption(specs.OptionNumLF) != "" {
		nlfopt, _ := strconv.Atoi(e.GetOption(specs.OptionNumLF))
		if nlfopt > 1 {
			task.NLF = nlfopt
			logger.Debug(fmt.Sprintf("[%s] Using %d LF on suffix", e.Endpoint, task.NLF))
		}
	}

	// Retrieve banner_visible option. Set to false when != "true"
	bannerVisible := e.GetOption(specs.OptionBannerVisible)
	if bannerVisible != "" && bannerVisible != "true" {
		task.BannerShow = false
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
			e.Endpoint, task.TermHeight, task.TermWidth, task.DeadlineSec, task.WaitMs))

		session, err = e.GetShellSession(e.Endpoint, term, task.TermHeight, task.TermWidth, disableEchoShell)
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

		if task.BannerLines > 0 {

			banner := ""

			for i := 0; i < task.BannerLines; i++ {

				line, err := session.stdoutPipeBuf.ReadString('\n')
				if err != nil {
					return 1, fmt.Errorf("read error: %w", err)
				}

				banner += line
			}

			logger.Debug(fmt.Sprintf("[%s] Skipped banner:\n%s---",
				e.Endpoint, banner))

			if task.BannerShow {
				output += banner
			}
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

		err = e.ciscoValidatePrompt(nodeName, session, task)
		if err != nil {
			return 1, err
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
			err = e.ciscoValidateEnaPrompt(nodeName, session, task)
			if err != nil {
				return 1, err
			}

			session.InEna = true
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
	_, err = session.stdinPipe.Write([]byte(command + suffix))

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

	waitMsDuration, _ := time.ParseDuration(fmt.Sprintf("%dms", task.WaitMs))
	i := 1
	for {
		if time.Now().After(deadline) {
			break
		}

		if i > 2 && task.FirstLine {
			logger.Debug(fmt.Sprintf("[%s] First line never skipped."), e.Endpoint)
		}

		line, err := session.stdoutPipeBuf.ReadString('\n')
		if err != nil && err != io.EOF {
			return 1, fmt.Errorf("read error: %w", err)
		}

		if task.FirstLine {
			// Skip first line with the written command.
			if e.ciscoIsCommandEchoLine(command, line, task, session) {
				task.FirstLine = false
				continue
			}

		}

		output += line

		if e.ciscoIsPromptLine(line, task, session) {
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
