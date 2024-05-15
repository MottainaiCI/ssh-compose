/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	helpers "github.com/MottainaiCI/ssh-compose/pkg/helpers"
	log "github.com/MottainaiCI/ssh-compose/pkg/logger"
	"github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/google/uuid"
)

func (e *SshCExecutor) RunCommandWithOutput(nodeName, command string, envs map[string]string, outBuffer, errBuffer io.WriteCloser, entryPoint []string) (int, error) {
	if outBuffer == nil {
		return 1, errors.New("Invalid outBuffer")
	}
	if errBuffer == nil {
		return 1, errors.New("Invalid errBuffer")
	}

	sid := uuid.New().String()

	session, err := e.GetSession(sid)
	if err != nil {
		return 1, fmt.Errorf("error on get session")
	}
	defer e.RemoveSession(sid)

	logger := log.GetDefaultLogger()

	if len(entryPoint) > 0 {
		e.Emitter.DebugLog(true, logger.Aurora.Bold(
			logger.Aurora.BrightCyan(
				fmt.Sprintf(">>> [%s] - entrypoint: %s", nodeName, entryPoint))))
	}

	e.Emitter.InfoLog(true, logger.Aurora.Italic(
		logger.Aurora.BrightCyan(
			fmt.Sprintf(">>> [%s] - %s - :coffee:", nodeName, command))))

	if len(envs) > 0 {
		keys := []string{}
		sshcproject := ""

		for k, _ := range envs {
			keys = append(keys, k)
		}

		sort.Strings(keys)
		for i, k := range keys {
			v, _ := envs[k]
			sshcproject += fmt.Sprintf("\"%s\": \"%s\"", k, v)
			if i < len(keys)-1 {
				sshcproject += ","
			}
		}

		_ = session.Setenv("SSH_COMPOSE_PROJECT", sshcproject)
		_ = session.Setenv("SSH_COMPOSE_VERSION", specs.SSH_COMPOSE_VERSION)
	}

	// Disable stdin
	session.Stdin = io.NopCloser(bytes.NewReader(nil))
	session.Stdout = outBuffer
	session.Stderr = errBuffer

	runArgs := ""
	if len(entryPoint) > 0 {
		runArgs = strings.Join(entryPoint, " ")
		runArgs += fmt.Sprintf("'%s'", command)
	} else {
		runArgs = command
	}

	ans := 0
	err = session.Run(runArgs)
	if err != nil {
		e.Emitter.InfoLog(true,
			logger.Aurora.Bold(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] Execution Interrupted: %s",
						nodeName, err.Error()))))
		ans = 1
	} else {
		e.Emitter.DebugLog(true,
			logger.Aurora.Bold(
				logger.Aurora.BrightCyan(
					fmt.Sprintf(">>> [%s] Exiting", nodeName, ans))))
	}

	return ans, nil
}

func (e *SshCExecutor) RunCommand(nodeName, command string, envs map[string]string, entryPoint []string) (int, error) {
	var outBuffer, errBuffer bytes.Buffer
	logger := log.GetDefaultLogger()

	res, err := e.RunCommandWithOutput(nodeName, command, envs,
		helpers.NewNopCloseWriter(&outBuffer), helpers.NewNopCloseWriter(&errBuffer),
		entryPoint)

	if err == nil {

		if e.ShowCmdsOutput && len(outBuffer.String()) > 0 && !e.RuntimeCmdsOutput {
			e.Emitter.InfoLog(false,
				logger.Aurora.Bold(
					logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] [stdout]\n%s", nodeName, outBuffer.String()))))
		}

		if e.ShowCmdsOutput && len(errBuffer.String()) > 0 && !e.RuntimeCmdsOutput {
			e.Emitter.InfoLog(false,
				logger.Aurora.Bold(
					logger.Aurora.BrightRed(
						fmt.Sprintf(">>> [%s] [stderr]\n%s", nodeName, errBuffer.String()))))
		}
	}

	return res, err
}

func (e *SshCExecutor) RunCommandWithOutput4Var(nodeName, command, outVar, errVar string, envs *map[string]string, entryPoint []string) (int, error) {
	var outBuffer, errBuffer bytes.Buffer
	logger := log.GetDefaultLogger()

	res, err := e.RunCommandWithOutput(nodeName, command, *envs,
		helpers.NewNopCloseWriter(&outBuffer), helpers.NewNopCloseWriter(&errBuffer),
		entryPoint)

	if err == nil {

		if e.ShowCmdsOutput && len(outBuffer.String()) > 0 {
			e.Emitter.InfoLog(false,
				logger.Aurora.Bold(
					logger.Aurora.BrightCyan(
						fmt.Sprintf(">>> [%s] [stdout]\n%s", nodeName, outBuffer.String()))))
		}

		if e.ShowCmdsOutput && len(errBuffer.String()) > 0 {
			e.Emitter.InfoLog(false,
				logger.Aurora.Bold(
					logger.Aurora.BrightRed(
						fmt.Sprintf(">>> [%s] [stderr]\n%s", nodeName, errBuffer.String()))))
		}

		if outVar != "" {
			(*envs)[outVar] = outBuffer.String()
		}
		if errVar != "" {
			(*envs)[errVar] = errBuffer.String()
		}
	}

	return res, err
}
