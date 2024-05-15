/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.

Based on the lxd-compose code
*/
package template

import (
	"os"
	"path/filepath"

	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
)

type MottainaiCompiler struct {
	*DefaultCompiler
}

func NewMottainaiCompiler(proj *specs.SshCProject) *MottainaiCompiler {
	return &MottainaiCompiler{
		DefaultCompiler: &DefaultCompiler{
			Project: proj,
		},
	}
}

func (r *MottainaiCompiler) Compile(sourceFile, destFile string) error {

	sourceData, err := os.ReadFile(sourceFile)
	if err != nil {
		return err
	}

	dstData, err := r.CompileRaw(string(sourceData))
	if err != nil {
		return err
	}

	dir := filepath.Dir(destFile)
	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	err = os.WriteFile(destFile, []byte(dstData), 0644)
	if err != nil {
		return err
	}

	return nil
}

func (r *MottainaiCompiler) CompileRaw(sourceData string) (string, error) {
	tmpl := NewTemplate()
	tmpl.Values = r.Vars

	destData, err := tmpl.Draw(sourceData)
	if err != nil {
		return "", err
	}

	return destData, nil
}
