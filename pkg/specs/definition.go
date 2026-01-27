/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

import (
	tarf_specs "github.com/geaaru/tar-formers/pkg/specs"
)

type SshCEnvironment struct {
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	File    string `json:"-" yaml:"-"`

	TemplateEngine SshCTemplateEngine `json:"template_engine,omitempty" yaml:"template_engine,omitempty"`

	Projects []SshCProject `json:"projects" yaml:"projects"`

	Commands             []SshCCommand `json:"commands,omitempty" yaml:"commands,omitempty"`
	IncludeCommandsFiles []string      `json:"include_commands_files,omitempty" yaml:"include_commands_files,omitempty"`

	PackExtra *SshCPackExtra `json:"pack_extra,omitempty" yaml:"pack_extra,omitempty"`
}

type SshCPackExtra struct {
	Dirs   []string                 `json:"dirs,omitempty" yaml:"dirs,omitempty"`
	Files  []string                 `json:"files,omitempty" yaml:"files,omitempty"`
	Rename []*tarf_specs.RenameRule `json:"rename,omitempty" yaml:"rename,omitempty"`
}

type SshCHook struct {
	Event      string   `json:"event" yaml:"event"`
	Node       string   `json:"node" yaml:"node"`
	Commands   []string `json:"commands,omitempty" yaml:"commands,omitempty"`
	Out2Var    string   `json:"out2var,omitempty" yaml:"out2var,omitempty"`
	Err2Var    string   `json:"err2var,omitempty" yaml:"err2var,omitempty"`
	Entrypoint []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Flags      []string `json:"flags,omitempty" yaml:"flags,omitempty"`
	Disable    bool     `json:"disable,omitempty" yaml:"disable,omitempty"`

	// Cisco specific flags
	CiscoEna bool `json:"cisco_ena,omitempty" yaml:"cisco_ena,omitempty"`

	// Pull resources
	PullResources      []*SshCSyncResource `json:"pull,omitempty" yaml:"pull,omitempty"`
	PullKeepSourcePath bool                `json:"pull_keep_sourcepath,omitempty" yaml:"pull_keep_sourcepath,omitempty"`
}

type SshCHooks struct {
	Hooks []SshCHook `json:"hooks,omitempty" yaml:"hooks,omitempty"`
}

type SshCTemplateEngine struct {
	Engine string   `json:"engine" yaml:"engine"`
	Opts   []string `json:"opts,omitempty" yaml:"opts,omitempty"`
}

type SshCProject struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	IncludeGroupFiles []string       `json:"include_groups_files,omitempty" yaml:"include_groups_files,omitempty"`
	IncludeEnvFiles   []string       `json:"include_env_files,omitempty" yaml:"include_env_files,omitempty"`
	IncludeHooksFiles []*SshCInclude `json:"include_hooks_files,omitempty" yaml:"include_hooks_files,omitempty"`

	Environments []SshCEnvVars `json:"vars,omitempty" yaml:"vars,omitempty"`

	Groups []SshCGroup `json:"groups" yaml:"groups"`

	Hooks           []SshCHook           `json:"hooks" yaml:"hooks"`
	ConfigTemplates []SshCConfigTemplate `json:"config_templates,omitempty" yaml:"config_templates,omitempty"`
}

type SshCProjectSanitized struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	IncludeGroupFiles []string       `json:"include_groups_files,omitempty" yaml:"include_groups_files,omitempty"`
	IncludeEnvFiles   []string       `json:"include_env_files,omitempty" yaml:"include_env_files,omitempty"`
	IncludeHooksFiles []*SshCInclude `json:"include_hooks_files,omitempty" yaml:"include_hooks_files,omitempty"`

	Groups []SshCGroup `json:"groups" yaml:"groups"`

	Hooks           []SshCHook           `json:"hooks" yaml:"hooks"`
	ConfigTemplates []SshCConfigTemplate `json:"config_templates,omitempty" yaml:"config_templates,omitempty"`
}

type SshCGroup struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Connection  string `json:"connection,omitempty" yaml:"connection,omitempty"`

	CommonProfiles []string          `json:"common_profiles,omitempty" yaml:"common_profiles,omitempty"`
	Config         map[string]string `json:"config,omitempty" yaml:"config,omitempty"`

	Ephemeral bool `json:"ephemeral,omitempty" yaml:"ephemeral,omitempty"`

	Nodes []SshCNode `json:"nodes" yaml:"nodes"`

	Hooks             []SshCHook           `json:"hooks" yaml:"hooks"`
	IncludeHooksFiles []*SshCInclude       `json:"include_hooks_files,omitempty" yaml:"include_hooks_files,omitempty"`
	ConfigTemplates   []SshCConfigTemplate `json:"config_templates,omitempty" yaml:"config_templates,omitempty"`
}

type SshCEnvVars struct {
	EnvVars map[string]interface{} `json:"envs,omitempty" yaml:"envs,omitempty"`
}

type SshCNode struct {
	Name string `json:"name" yaml:"name"`

	Endpoint string `json:"endpoint,omitempty" yaml:"endpoint,omitempty"`

	Labels map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	Config map[string]string `json:"config,omitempty" yaml:"config,omitempty"`

	SourceDir string `json:"source_dir,omitempty" yaml:"source_dir,omitempty"`

	Entrypoint []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`

	ConfigTemplates []SshCConfigTemplate `json:"config_templates,omitempty" yaml:"config_templates,omitempty"`
	SyncResources   []SshCSyncResource   `json:"sync_resources,omitempty" yaml:"sync_resources,omitempty"`

	Hooks             []SshCHook     `json:"hooks" yaml:"hooks"`
	IncludeHooksFiles []*SshCInclude `json:"include_hooks_files,omitempty" yaml:"include_hooks_files,omitempty"`
}

type SshCConfigTemplate struct {
	Source      string `json:"source" yaml:"source"`
	Destination string `json:"dst" yaml:"dst"`
}

type SshCSyncResource struct {
	Source      string `json:"source" yaml:"source"`
	Destination string `json:"dst" yaml:"dst"`
}

type SshCCommand struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Project     string `json:"project" yaml:"project"`
	ApplyAlias  bool   `json:"apply_alias,omitempty" yaml:"apply_alias,omitempty"`

	SkipSync    bool `json:"skip_sync,omitempty" yaml:"skip_sync,omitempty"`
	SkipCompile bool `json:"skip_compile,omitempty" yaml:"skip_compile,omitempty"`
	Destroy     bool `json:"destroy,omitempty" yaml:"destroy,omitempty"`

	EnableFlags  []string `json:"enable_flags,omitempty" yaml:"enable_flags,omitempty"`
	DisableFlags []string `json:"disable_flags,omitempty" yaml:"disable_flags,omitempty"`

	EnableGroups  []string `json:"enable_groups,omitempty" yaml:"enable_groups,omitempty"`
	DisableGroups []string `json:"disable_groups,omitempty" yaml:"disable_groups,omitempty"`

	Envs     SshCEnvVars `json:"envs,omitempty" yaml:"envs,omitempty"`
	VarFiles []string    `json:"vars_files,omitempty" yaml:"vars_files,omitempty"`

	IncludeHooksFiles []*SshCInclude `json:"include_hooks_files,omitempty" yaml:"include_hooks_files,omitempty"`
}

type SshCInclude struct {
	Type  string   `json:"mode,omitempty" yaml:"mode,omitempty"`
	Files []string `json:"files,omitempty" yaml:"files,omitempty"`
}
