/*
Copyright © 2024-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cisco

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/MottainaiCI/ssh-compose/pkg/executor/core"
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"
)

const (
	CiscoProfileTypeGeneral = "general"
	CiscoProfileTypeAsaFw4K = "asa-fw-4k"
)

type CiscoExecTask struct {
	CiscoPromptMatchRegex    *regexp.Regexp
	CiscoPromptEnaMatchRegex *regexp.Regexp
	FirstLine                bool
	Session                  *core.SshCSession
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

type CiscoProfile interface {
	ValidatePrompt(nodeName string, emitter core.SshCExecutorEmitter, session *core.SshCSession, task *CiscoExecTask) error
	ValidateEnaPrompt(nodeName string, emitter core.SshCExecutorEmitter, session *core.SshCSession, task *CiscoExecTask) error
	IsCommandEchoLine(command, line string, task *CiscoExecTask, session *core.SshCSession) bool
	IsPromptLine(line string, task *CiscoExecTask, session *core.SshCSession) bool
	Login(nodeName string, emitter core.SshCExecutorEmitter, task *CiscoExecTask,
		session *core.SshCSession, opts *core.CiscoCommandOpts) (string, error)

	GetTask(nodeName string, options *map[string]string, opts *core.CiscoCommandOpts) (*CiscoExecTask, error)
	GetType() string
}

type CiscoProfileBase struct {
	// Cisco Device options
	CiscoPromptRegex bool
	CiscoPrompt      string
	CiscoEnaPrompt   string
	CiscoEnaPass     string
}

func newCiscoProfileBase(promptWithRegex bool, prompt, enaPrompt, enaPass string) *CiscoProfileBase {
	return &CiscoProfileBase{
		CiscoPromptRegex: promptWithRegex,
		CiscoPrompt:      prompt,
		CiscoEnaPrompt:   enaPrompt,
		CiscoEnaPass:     enaPass,
	}
}

func NewCiscoProfile(pname, prompt, enaPrompt, enaPass string, promptWithRegex bool) CiscoProfile {
	switch pname {
	case CiscoProfileTypeAsaFw4K:
		return NewCiscoProfileAsaFw4K(promptWithRegex, prompt, enaPrompt, enaPass)
	default:
		return NewCiscoProfileGeneral(promptWithRegex, prompt, enaPrompt, enaPass)
	}
}

func getOption(options *map[string]string, o string) string {
	if options != nil {
		for k, v := range *options {
			if k == o {
				return v
			}
		}
	}
	return ""
}

func (p *CiscoProfileBase) setTask(nodeName string,
	task *CiscoExecTask, options *map[string]string, opts *core.CiscoCommandOpts) {

	logger := log.GetDefaultLogger()

	// Retrieve height and width from remote option
	height := getOption(options, specs.OptionTermHeight)
	if height != "" {
		oh, _ := strconv.Atoi(height)
		if oh > 0 {
			task.TermHeight = oh
		}
	}
	width := getOption(options, specs.OptionTermWidth)
	if width != "" {
		ow, _ := strconv.Atoi(width)
		if ow > 0 {
			task.TermWidth = ow
		}
	}
	// Retrieve deadline_secs from remote option
	dls := getOption(options, specs.OptionDeadlineSecs)
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
	waitms := getOption(options, specs.OptionWaitMs)
	if waitms != "" {
		owms, _ := strconv.Atoi(waitms)
		if owms > 0 {
			task.WaitMs = owms
		}
	}
	// Retrieve banner lines option
	if getOption(options, specs.OptionBannerLines) != "" {
		blines, _ := strconv.Atoi(getOption(options, specs.OptionBannerLines))
		if blines > 0 {
			task.BannerLines = blines
		}
	}

	// Retrieve number of LF (line feed) to append in the command.
	if getOption(options, specs.OptionNumLF) != "" {
		nlfopt, _ := strconv.Atoi(getOption(options, specs.OptionNumLF))
		if nlfopt > 1 {
			task.NLF = nlfopt
			logger.Debug(fmt.Sprintf("[%s] Using %d LF on suffix", nodeName, task.NLF))
		}
	}

	// Retrieve banner_visible option. Set to false when != "true"
	bannerVisible := getOption(options, specs.OptionBannerVisible)
	if bannerVisible != "" && bannerVisible != "true" {
		task.BannerShow = false
	}
}

func (p *CiscoProfileBase) ValidatePrompt(nodeName string, emitter core.SshCExecutorEmitter,
	session *core.SshCSession, task *CiscoExecTask) error {

	logger := log.GetDefaultLogger()

	if p.CiscoPrompt != "" && (!p.CiscoPromptRegex) && session.CiscoPrompt != p.CiscoPrompt {
		// POST: CiscoPrompt is defined but is not a regex.
		emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) != '%s' (config)",
			nodeName, session.CiscoPrompt, p.CiscoPrompt))
	} else if p.CiscoPromptRegex {
		// POST: CiscoPromptRegex is enable.

		if p.CiscoPrompt == "" {
			return fmt.Errorf("invalid configuration with empty prompt string and regex enabled")
		}

		// Compile and validate Regex
		task.CiscoPromptMatchRegex = regexp.MustCompile(p.CiscoPrompt)
		if task.CiscoPromptMatchRegex == nil {
			return fmt.Errorf("failed on compile cisco prompt regex from value %s", p.CiscoPrompt)
		}

		if task.CiscoPromptMatchRegex.MatchString(session.CiscoPrompt) {
			logger.Debug(fmt.Sprintf("[%s] %s regex match %s", nodeName, p.CiscoPrompt, session.CiscoPrompt))
		} else {
			emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on prompt '%s' (session) for regex '%s' (config)",
				nodeName, session.CiscoPrompt, p.CiscoPrompt))
		}

	} else if p.CiscoPrompt == "" {
		emitter.InfoLog(true, logger.Aurora.Bold(
			logger.Aurora.Italic(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - Using cisco prompt %s - :eye:", nodeName, session.CiscoPrompt)))))
	}

	return nil
}

func (p *CiscoProfileBase) ValidateEnaPrompt(nodeName string, emitter core.SshCExecutorEmitter,
	session *core.SshCSession, task *CiscoExecTask) error {

	logger := log.GetDefaultLogger()

	if p.CiscoEnaPrompt != "" && (!p.CiscoPromptRegex) && session.CiscoEnaPrompt != p.CiscoEnaPrompt {
		emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on ena prompt '%s' (session) != '%s' (config)",
			nodeName, session.CiscoEnaPrompt, p.CiscoEnaPrompt))

	} else if p.CiscoPromptRegex {
		// POST: CiscoPromptRegex is enable.

		if p.CiscoEnaPrompt == "" {
			return fmt.Errorf("invalid configuration with empty ena prompt string and regex enabled")
		}

		// Compile and validate Regex
		task.CiscoPromptEnaMatchRegex = regexp.MustCompile(p.CiscoEnaPrompt)
		if task.CiscoPromptEnaMatchRegex == nil {
			return fmt.Errorf("failed on compile cisco ena prompt regex from value %s", p.CiscoEnaPrompt)
		}

		if task.CiscoPromptEnaMatchRegex.MatchString(session.CiscoEnaPrompt) {
			logger.Debug(fmt.Sprintf("[%s] %s regex match %s", nodeName, p.CiscoEnaPrompt, session.CiscoEnaPrompt))
		} else {
			emitter.WarnLog(false, fmt.Sprintf("[%s] Mismatch on ena prompt '%s' (session) for regex '%s' (config)",
				nodeName, session.CiscoEnaPrompt, p.CiscoEnaPrompt))
		}

	} else if p.CiscoEnaPrompt == "" {
		emitter.InfoLog(true, logger.Aurora.Bold(
			logger.Aurora.Italic(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] - Using cisco ena prompt %s - :eye:", nodeName, session.CiscoEnaPrompt)))))
	}

	return nil
}

func (p *CiscoProfileBase) IsCommandEchoLine(command, line string,
	task *CiscoExecTask, session *core.SshCSession) bool {

	ans := false

	lineTrim := strings.TrimSpace(line)
	if strings.HasPrefix(lineTrim, command) {
		ans = true
	} else if p.CiscoPromptRegex {

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

func (p *CiscoProfileBase) IsPromptLine(line string, task *CiscoExecTask, session *core.SshCSession) bool {
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

	if p.CiscoPromptRegex {

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
