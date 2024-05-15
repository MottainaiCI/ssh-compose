/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"github.com/MottainaiCI/ssh-compose/pkg/helpers"
)

func GetSshCRemotesDefaultConfDir() (string, error) {
	var configDir string

	if os.Getenv("SSHC_CONF") != "" {
		configDir = os.Getenv("SSHC_CONF")
	} else if os.Getenv("HOME") != "" {
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ssh-compose")
	} else {
		user, err := user.Current()
		if err != nil {
			return "", err
		}

		configDir = filepath.Join(user.HomeDir, ".config", "ssh-compose")
	}

	return configDir, nil
}

func LoadRemotesConfig(remotesconfdir string) (*RemotesConfig, error) {
	var ans *RemotesConfig
	var err error

	if remotesconfdir != "" {
		remotesconf := filepath.Join(
			remotesconfdir, "config.yml",
		)

		// Check if exists config.yml
		if helpers.Exists(remotesconf) {
			ans, err = RemotesConfigFromFile(remotesconf)
			if err != nil {
				return ans, err
			}
		} else {
			return ans, fmt.Errorf(
				"the file %s doesn't exist.",
				remotesconf)
		}
	} else {
		sshcDir, err := GetSshCRemotesDefaultConfDir()
		if err != nil {
			return ans, err
		}

		remotesconf := filepath.Join(sshcDir, "config.yml")
		if helpers.Exists(remotesconf) {
			ans, err = RemotesConfigFromFile(remotesconf)
			if err != nil {
				return ans, err
			}
		} else {
			ans = NewRemotesConfig()
			ans.File = remotesconf
		}
	}
	return ans, nil
}
