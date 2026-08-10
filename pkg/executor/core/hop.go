/*
Copyright © 2024-2026 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package core

import (
	"os"

	"github.com/MottainaiCI/ssh-compose/pkg/specs"
	"golang.org/x/crypto/ssh"
)

type TunnelHop struct {
	// Ssh connection protocol. Valid values: tcp,tcp4,tcp6,unix
	ConnProtocol string
	Host         string
	Port         int
	TimeoutSecs  *uint

	User           string
	Pass           string
	PrivateKey     string
	PrivateKeyPass string

	Client *ssh.Client
}

func NewTunnelHop(r *specs.Remote) (*TunnelHop, error) {
	ans := &TunnelHop{
		ConnProtocol: r.Protocol,
		User:         r.User,
		Host:         r.Host,
		Port:         r.Port,
		TimeoutSecs:  r.TimeoutSecs,
	}

	if r.AuthMethod == specs.AuthMethodPassword {
		ans.Pass = r.Pass
	} else {
		ans.PrivateKeyPass = r.PrivateKeyPass

		if r.PrivateKeyFile != "" {
			data, err := os.ReadFile(r.PrivateKeyFile)
			if err != nil {
				return ans, err
			}

			ans.PrivateKey = string(data)
		} else {
			ans.PrivateKey = r.PrivateKeyRaw
		}
	}

	return ans, nil
}
