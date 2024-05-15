/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.

Based on the lxd-compose code
*/
package template

import (
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
)

type SshCTemplateCompiler interface {
	InitVars()
	SetOpts([]string)
	Compile(sourceFile, destFile string) error
	CompileRaw(sourceContent string) (string, error)
	GetEnvBaseDir() string
	SetEnvBaseDir(string)
	GetVars() *map[string]interface{}
}

type DefaultCompiler struct {
	Project    *specs.SshCProject
	Opts       []string
	Vars       map[string]interface{}
	EnvBaseDir string
}

func (r *DefaultCompiler) InitVars() {
	r.Vars = make(map[string]interface{}, 0)
	for _, evar := range r.Project.Environments {
		for k, v := range evar.EnvVars {
			r.Vars[k] = v
		}
	}
	// Init project variable
	r.Vars["project"] = r.Project
	r.Vars["env_base_dir"] = r.EnvBaseDir
}

func (r *DefaultCompiler) SetOpts(o []string) {
	r.Opts = o
}

func (r *DefaultCompiler) GetEnvBaseDir() string {
	return r.EnvBaseDir
}

func (r *DefaultCompiler) SetEnvBaseDir(dir string) {
	r.EnvBaseDir = dir
}

func (r *DefaultCompiler) GetVars() *map[string]interface{} {
	return &r.Vars
}
