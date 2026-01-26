/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	cliName = `Copyright © 2024-2026 Mottainai - Daniele Rondina

Mottainai - SSH Compose Integrator`
)

var (
	BuildTime      string
	BuildCommit    string
	BuildGoVersion string
)

func initConfig(config *specs.SshComposeConfig) {
	// Set env variable
	config.Viper.SetEnvPrefix(specs.SSH_COMPOSE_ENV_PREFIX)
	config.Viper.BindEnv("config")
	config.Viper.SetDefault("config", "")
	config.Viper.SetDefault("etcd-config", false)

	config.Viper.AutomaticEnv()

	// Create EnvKey Replacer for handle complex structure
	replacer := strings.NewReplacer(".", "__")
	config.Viper.SetEnvKeyReplacer(replacer)

	// Set config file name (without extension)
	config.Viper.SetConfigName(specs.SSH_COMPOSE_CONFIGNAME)

	config.Viper.SetTypeByDefaultValue(true)

}

func cmdNeedConfig(cmd *cobra.Command) bool {
	ans := true
	commandWorksWithoutConfig := []string{
		"shell",
		"remote",
		"exec",
		"file",
	}

	name := cmd.Name()
	if name != "" {
		if cmd.Parent().Name() != "" {
			name = cmd.Parent().Name()
		}
	}

	for _, c := range commandWorksWithoutConfig {
		if c == name {
			ans = false
			break
		}
	}

	return ans
}

func initCommand(rootCmd *cobra.Command, config *specs.SshComposeConfig) {
	var pflags = rootCmd.PersistentFlags()

	pflags.StringP("config", "c", "", "SSH Compose configuration file")
	pflags.String("render-values", "", "Override render values file.")
	pflags.String("render-default", "", "Override render default file.")
	pflags.Bool("cmds-output", config.Viper.GetBool("logging.cmds_output"),
		"Show hooks commands output or not.")
	pflags.BoolP("debug", "d", config.Viper.GetBool("general.debug"),
		"Enable debug output.")

	config.Viper.BindPFlag("config", pflags.Lookup("config"))
	config.Viper.BindPFlag("render_default_file", pflags.Lookup("render-default"))
	config.Viper.BindPFlag("render_values_file", pflags.Lookup("render-values"))
	config.Viper.BindPFlag("general.debug", pflags.Lookup("debug"))
	config.Viper.BindPFlag("logging.cmds_output", pflags.Lookup("cmds-output"))

	rootCmd.AddCommand(
		NewCompileCommand(config),
		NewShellCommand(config),
		newCommandCommand(config),
		NewExecCommand(config),
		newProjectCommand(config),
		newRemoteCommand(config),
		newGroupCommand(config),
		newApplyCommand(config),
		newFileCommand(config),
		newValidateCommand(config),
		newDiagnoseCommand(config),
		NewTunnelCommand(config),
	)
}

func version() string {
	ans := fmt.Sprintf("%s-g%s %s", specs.SSH_COMPOSE_VERSION,
		BuildCommit, BuildTime)
	if BuildGoVersion != "" {
		ans += " " + BuildGoVersion
	}
	return ans
}

func Execute() {
	// Create Main Instance Config object
	var config *specs.SshComposeConfig = specs.NewSshComposeConfig(nil)

	initConfig(config)

	var rootCmd = &cobra.Command{
		Short:        cliName,
		Version:      version(),
		Args:         cobra.OnlyValidArgs,
		SilenceUsage: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(0)
			}
		},
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			var err error
			var v *viper.Viper = config.Viper

			v.SetConfigType("yml")
			if v.Get("config") == "" {
				config.Viper.AddConfigPath(".")
			} else {
				v.SetConfigFile(v.Get("config").(string))
			}

			// Parse configuration file
			err = config.Unmarshal()
			if err != nil {
				if !cmdNeedConfig(cmd) {
					// Trying to loading defaults
					v.Unmarshal(&config)
				} else {
					fmt.Println(err.Error())
					os.Exit(1)
				}
			}
		},
	}

	initCommand(rootCmd, config)

	// Start command execution
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
