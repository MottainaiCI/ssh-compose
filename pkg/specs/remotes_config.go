/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
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

func LoadRemotesConfig(remotesconfdir string,
	config *SshComposeConfig) (*RemotesConfig, error) {
	var ans *RemotesConfig
	var err error

	if remotesconfdir != "" {
		remotesconf := filepath.Join(
			remotesconfdir, "config.yml",
		)

		// Check if exists config.yml
		if helpers.Exists(remotesconf) {
			ans, err = RemotesConfigFromFile(remotesconf, config)
			if err != nil {
				return ans, err
			}
		} else {
			if !helpers.Exists(remotesconf) {
				os.MkdirAll(remotesconfdir, os.ModePerm)
			}
			// Just initialize an empty struct if the file doesn't
			// exist. This could be happens in the initial setup.
			ans = NewRemotesConfig()
			ans.File = remotesconf
		}
	} else {
		sshcDir, err := GetSshCRemotesDefaultConfDir()
		if err != nil {
			return ans, err
		}

		remotesconf := filepath.Join(sshcDir, "config.yml")
		if helpers.Exists(remotesconf) {
			ans, err = RemotesConfigFromFile(remotesconf, config)
			if err != nil {
				return ans, err
			}
		} else {
			// Create directory if not present
			if !helpers.Exists(sshcDir) {
				os.MkdirAll(sshcDir, os.ModePerm)
			}

			ans = NewRemotesConfig()
			ans.File = remotesconf
		}
	}

	// Correctly setup empty files.
	if ans.Remotes == nil {
		ans.Remotes = make(map[string]*Remote, 0)
		ans.DefaultRemote = ""
	}
	return ans, nil
}
