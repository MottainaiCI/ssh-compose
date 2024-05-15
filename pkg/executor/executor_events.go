/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package executor

const (
	SshClientSetupDone    SshCExecutorEvent = "client-setup"
	SshContainerConnected SshCExecutorEvent = "endpoint-connected"
)

type SshCExecutorEvent string

type SshCExecutorEmitter interface {
	Emits(eType SshCExecutorEvent, data map[string]interface{})

	DebugLog(color bool, args ...interface{})
	InfoLog(color bool, args ...interface{})
	WarnLog(color bool, args ...interface{})
	ErrorLog(color bool, args ...interface{})
}
