/*
Copyright © 2024-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cisco

import (
	"fmt"
	"strings"
	"time"

	"github.com/MottainaiCI/ssh-compose/pkg/executor/core"
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
)

type CiscoProfileAsaFw4K struct {
	*CiscoProfileBase
}

func NewCiscoProfileAsaFw4K(promptWithRegex bool, prompt, enaPrompt, enaPass string) *CiscoProfileAsaFw4K {
	return &CiscoProfileAsaFw4K{
		CiscoProfileBase: newCiscoProfileBase(promptWithRegex, prompt, enaPrompt, enaPass),
	}
}

func (p *CiscoProfileAsaFw4K) GetType() string { return CiscoProfileTypeAsaFw4K }

func (p *CiscoProfileAsaFw4K) GetTask(nodeName string,
	options *map[string]string, opts *core.CiscoCommandOpts) (*CiscoExecTask, error) {

	task := &CiscoExecTask{
		FirstLine:               true,
		TermHeight:              500,
		TermWidth:               200,
		DeadlineSec:             30,
		WaitMs:                  40,
		NLF:                     1,
		BannerLines:             0,
		BannerShow:              true,
		UsedPromptHasFinalSpace: false,
	}

	p.CiscoProfileBase.setTask(nodeName, task, options, opts)

	return task, nil
}

func (p *CiscoProfileAsaFw4K) Login(nodeName string, emitter core.SshCExecutorEmitter,
	task *CiscoExecTask, session *core.SshCSession, opts *core.CiscoCommandOpts) (string, error) {

	logger := log.GetDefaultLogger()

	output := ""
	waitMsDuration, _ := time.ParseDuration(fmt.Sprintf("%dms", task.WaitMs))
	buff := make([]byte, 1024)

	// Get time to device to write the prompt. Maybe could be
	// set in the remote config option.
	time.Sleep(1000 * time.Millisecond)

	if task.BannerLines > 0 {

		banner := ""

		for i := 0; i < task.BannerLines; i++ {

			line, err := session.CstdoutPipeBuf.ReadString('\n')
			if err != nil {
				return output, fmt.Errorf("read error: %w", err)
			}

			banner += line
		}

		logger.Debug(fmt.Sprintf("[%s] Skipped banner:\n%s---",
			nodeName, banner))

		if task.BannerShow {
			output += banner
		}
	}

	// Read prompt
	n, err := session.CstdoutPipeBuf.Read(buff)
	if err != nil {
		return output, fmt.Errorf("failed on read prompt: %v", err)
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

	output += string(buff[0:n])

	err = p.ValidatePrompt(nodeName, emitter, session, task)
	if err != nil {
		return output, err
	}

	if opts.WithEna {

		// POST: The command requires ena privileges

		// Send ena command on stdin
		_, err = session.CstdinPipe.Write([]byte("ena" + "\r"))
		if err != nil {
			return output, fmt.Errorf("failed to write ena command: %w", err)
		}

		// Read the sent command
		line, _ := session.CstdoutPipeBuf.ReadString('\n')

		output += line

		// Waiting a bit
		time.Sleep(waitMsDuration)

		// Read the ask password output
		n, _ := session.CstdoutPipeBuf.Read(buff)
		line = string(buff[0:n])

		output += line

		if !strings.Contains(line, "Password:") {
			return output, fmt.Errorf("received invalid response for ena command: %s", line)
		}

		// ASA Firepower 4K on sent wrong password just return again the
		// "Password:" .
		_, _ = session.CstdinPipe.Write([]byte(p.CiscoEnaPass + "\r"))

		// Waiting a bit else I don't read the response from device.
		time.Sleep(waitMsDuration)

		// Ignoring Response **** \r\n
		n, _ = session.CstdoutPipeBuf.Read(buff)

		output += string(buff[0:n])

		// Ignore the first two byte 13 (\r) and 10 (\n)
		if buff[0] == '\r' && buff[1] == '\n' {
			line = string(buff[2:n])
		} else if buff[0] == '\r' {
			line = string(buff[1:n])
		} else if buff[0] == '\n' {
			line = string(buff[1:n])
		} else {
			line = string(buff[0:n])
		}

		if strings.Contains(line, "Password:") {
			return output, fmt.Errorf("invalid ena credential")
		}

		// On ASA using sleep before, if the password is correct
		// the buffer contains both the line with the  * that the
		// line with the prompt. NOTE: The prompt is without the final \r.

		lines := strings.Split(line, "\n")
		if len(lines) > 1 {

			// Using last line only
			line = lines[len(lines)-1]

			if line[0:1] == "\r" {
				line = line[1:]
			}

		}

		// Firepower 4200 return Access denied.
		// ASA
		if strings.Contains(output, "Invalid password") || strings.Contains(output, "Access denied") {
			return output, fmt.Errorf("invalid ena credential")
		}

		// POST: if all works fine the line will contains the new prompt

		session.CiscoEnaPrompt = line
		err = p.ValidateEnaPrompt(nodeName, emitter, session, task)
		if err != nil {
			return output, err
		}

		session.InEna = true
	}

	return output, nil
}
