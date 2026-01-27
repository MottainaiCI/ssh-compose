/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	v "github.com/spf13/viper"

	"gopkg.in/yaml.v3"
)

const (
	SSH_COMPOSE_CONFIGNAME = ".ssh-compose"
	SSH_COMPOSE_ENV_PREFIX = "ssh_COMPOSE"
	SSH_COMPOSE_VERSION    = `0.9.0`
)

type SshComposeConfig struct {
	Viper *v.Viper `yaml:"-" json:"-"`

	General         SshCGeneral `mapstructure:"general" json:"general,omitempty" yaml:"general,omitempty"`
	Logging         SshCLogging `mapstructure:"logging" json:"logging,omitempty" yaml:"logging,omitempty"`
	EnvironmentDirs []string    `mapstructure:"env_dirs,omitempty" json:"env_dirs,omitempty" yaml:"env_dirs,omitempty"`

	RenderDefaultFile   string                 `mapstructure:"render_default_file,omitempty" json:"render_default_file,omitempty" yaml:"render_default_file,omitempty"`
	RenderValuesFile    string                 `mapstructure:"render_values_file,omitempty" json:"render_values_file,omitempty" yaml:"render_values_file,omitempty"`
	RenderEnvsVars      map[string]interface{} `mapstructure:"-" json:"-" yaml:"-"`
	RenderTemplatesDirs []string               `mapstructure:"render_templates_dirs,omitempty" json:"render_templates_dirs,omitempty" yaml:"render_templates_dirs,omitempty"`
}

type SshCGeneral struct {
	Debug            bool   `mapstructure:"debug,omitempty" json:"debug,omitempty" yaml:"debug,omitempty"`
	RemotesConfDir   string `mapstructure:"remotes_confdir,omitempty" json:"remotes_confdir,omitempty" yaml:"remotes_confdir,omitempty"`
	EnvSessionPrefix string `mapstructure:"env_session_prefix,omitempty" json:"env_session_prefix,omitempty" yaml:"env_session_prefix,omitempty"`
}

type SshCLogging struct {
	// Path of the logfile
	Path string `mapstructure:"path,omitempty" json:"path,omitempty" yaml:"path,omitempty"`
	// Enable/Disable logging to file
	EnableLogFile bool `mapstructure:"enable_logfile,omitempty" json:"enable_logfile,omitempty" yaml:"enable_logfile,omitempty"`
	// Enable JSON format logging in file
	JsonFormat bool `mapstructure:"json_format,omitempty" json:"json_format,omitempty" yaml:"json_format,omitempty"`

	// Log level
	Level string `mapstructure:"level,omitempty" json:"level,omitempty" yaml:"level,omitempty"`

	// Enable emoji
	EnableEmoji bool `mapstructure:"enable_emoji,omitempty" json:"enable_emoji,omitempty" yaml:"enable_emoji,omitempty"`
	// Enable/Disable color in logging
	Color bool `mapstructure:"color,omitempty" json:"color,omitempty" yaml:"color,omitempty"`

	// Enable/Disable commands output logging
	RuntimeCmdsOutput bool `mapstructure:"runtime_cmds_output,omitempty" json:"runtime_cmds_output,omitempty" yaml:"runtime_cmds_output,omitempty"`
	CmdsOutput        bool `mapstructure:"cmds_output,omitempty" json:"cmds_output,omitempty" yaml:"cmds_output,omitempty"`
}

func NewSshComposeConfig(viper *v.Viper) *SshComposeConfig {
	if viper == nil {
		viper = v.New()
	}

	GenDefault(viper)
	return &SshComposeConfig{Viper: viper}
}

func (c *SshComposeConfig) Clone() *SshComposeConfig {
	ans := NewSshComposeConfig(nil)

	ans.EnvironmentDirs = c.EnvironmentDirs
	ans.RenderDefaultFile = c.RenderDefaultFile
	ans.RenderValuesFile = c.RenderValuesFile
	ans.RenderTemplatesDirs = c.RenderTemplatesDirs

	ans.General.Debug = c.General.Debug

	ans.Logging.Path = c.Logging.Path
	ans.Logging.EnableLogFile = c.Logging.EnableLogFile
	ans.Logging.JsonFormat = c.Logging.JsonFormat
	ans.Logging.Level = c.Logging.Level
	ans.Logging.EnableEmoji = c.Logging.EnableEmoji
	ans.Logging.Color = c.Logging.Color
	ans.Logging.RuntimeCmdsOutput = c.Logging.RuntimeCmdsOutput
	ans.Logging.CmdsOutput = c.Logging.CmdsOutput

	return ans
}

func (c *SshComposeConfig) GetGeneral() *SshCGeneral {
	return &c.General
}

func (c *SshComposeConfig) GetEnvironmentDirs() []string {
	return c.EnvironmentDirs
}

func (c *SshComposeConfig) GetLogging() *SshCLogging {
	return &c.Logging
}

func (c *SshComposeConfig) IsEnableRenderEngine() bool {
	if c.RenderValuesFile != "" || c.RenderDefaultFile != "" {
		return true
	}
	return false
}

func (c *SshComposeConfig) Unmarshal() error {
	var err error

	if c.Viper.InConfig("etcd-config") &&
		c.Viper.GetBool("etcd-config") {
		err = c.Viper.ReadRemoteConfig()
	} else {
		err = c.Viper.ReadInConfig()
	}

	if err != nil {
		return err
	}

	err = c.Viper.Unmarshal(&c)

	return err
}

func (c *SshComposeConfig) Yaml() ([]byte, error) {
	return yaml.Marshal(c)
}

func (c *SshComposeConfig) SetRenderEnvs(envs []string) error {
	e := NewEnvVars()

	for _, env := range envs {
		err := e.AddKVAggregated(env)
		if err != nil {
			return err
		}
	}

	if len(e.EnvVars) > 0 {
		c.RenderEnvsVars = e.EnvVars
	}

	return nil
}

func GenDefault(viper *v.Viper) {
	viper.SetDefault("general.debug", false)
	viper.SetDefault("general.env_session_prefix", "SSH_COMPOSE")
	viper.SetDefault("render_default_file", "")
	viper.SetDefault("render_values_file", "")
	viper.SetDefault("render_templates_dirs", []string{})

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.enable_logfile", false)
	viper.SetDefault("logging.path", "./logs/ssh-compose.log")
	viper.SetDefault("logging.json_format", false)
	viper.SetDefault("logging.enable_emoji", true)
	viper.SetDefault("logging.color", true)
	viper.SetDefault("logging.cmds_output", true)
	viper.SetDefault("logging.runtime_cmds_output", true)
	viper.SetDefault("logging.push_progressbar", false)

	viper.SetDefault("env_dirs", []string{"./envs"})
}

func (g *SshCGeneral) HasDebug() bool {
	return g.Debug
}
