/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

func GetTemplates(templateDirs []string) ([]*chart.File, error) {
	var regexConfs = regexp.MustCompile(`.yaml$`)

	ans := []*chart.File{}
	for _, tdir := range templateDirs {
		dirEntries, err := os.ReadDir(tdir)
		if err != nil {
			return ans, err
		}

		for _, file := range dirEntries {
			if file.IsDir() {
				continue
			}

			if !regexConfs.MatchString(file.Name()) {
				continue
			}

			content, err := os.ReadFile(path.Join(tdir, file.Name()))
			if err != nil {
				return ans, fmt.Errorf(
					"Error on read template file %s/%s: %s",
					tdir, file.Name(), err.Error())
			}

			ans = append(ans, &chart.File{
				// Using filename without extension for chart file name
				Name: strings.ReplaceAll(file.Name(), ".yaml", ""),
				Data: content,
			})

		}
	}

	return ans, nil
}

func RenderContentWithTemplates(
	raw, valuesFile, defaultFile, originFile string,
	overrideValues map[string]interface{},
	templateDirs []string) (string, error) {

	var err error

	if valuesFile == "" && defaultFile == "" {
		return "", errors.New("Both render files are missing")
	}

	values := make(map[string]interface{}, 0)
	d := make(map[string]interface{}, 0)

	if valuesFile != "" {
		if !Exists(valuesFile) {
			return "", errors.New(fmt.Sprintf(
				"Render value file %s not existing ", valuesFile))
		}
		val, err := os.ReadFile(valuesFile)
		if err != nil {
			return "", errors.New(fmt.Sprintf(
				"Error on reading Render value file %s: %s", valuesFile, err.Error()))
		}

		if err = yaml.Unmarshal(val, &values); err != nil {
			return "", errors.New(fmt.Sprintf(
				"Error on unmarsh file %s: %s", valuesFile, err.Error()))
		}
	}

	if defaultFile != "" {
		if !Exists(defaultFile) {
			return "", errors.New(fmt.Sprintf(
				"Render value file %s not existing ", defaultFile))
		}

		def, err := os.ReadFile(defaultFile)
		if err != nil {
			return "", errors.New(fmt.Sprintf(
				"Error on reading Render value file %s: %s", valuesFile, err.Error()))
		}

		if err = yaml.Unmarshal(def, &d); err != nil {
			return "", errors.New(fmt.Sprintf(
				"Error on unmarshal file %s: %s", defaultFile, err.Error()))
		}
	}

	if len(overrideValues) > 0 {
		for k, v := range overrideValues {
			values[k] = v
		}
	}

	charts := []*chart.File{}
	if len(templateDirs) > 0 {
		charts, err = GetTemplates(templateDirs)
		if err != nil {
			return "", err
		}
	}

	charts = append(charts, &chart.File{
		Name: "templates",
		Data: []byte(raw),
	})

	c := &chart.Chart{
		Metadata: &chart.Metadata{
			Name:    "tpl",
			Version: "",
		},
		Templates: charts,
		Values:    map[string]interface{}{"Values": d},
	}

	v, err := chartutil.CoalesceValues(c, map[string]interface{}{"Values": values})
	if err != nil {
		return "", errors.New(fmt.Sprintf(
			"Error on coalesce values for file %s: %s", originFile, err.Error()))
	}
	out, err := engine.Render(c, v)
	if err != nil {
		return "", errors.New(fmt.Sprintf(
			"Error on rendering file %s: %s", originFile, err.Error()))
	}

	debugHelmTemplate := os.Getenv("SSH_COMPOSE_HELM_DEBUG")
	if debugHelmTemplate == "1" {
		fmt.Println(out["tpl/templates"])
	}

	return out["tpl/templates"], nil
}

func RenderContent(raw, valuesFile, defaultFile, originFile string,
	overrideValues map[string]interface{}) (string, error) {

	return RenderContentWithTemplates(raw, valuesFile, defaultFile, originFile,
		overrideValues, []string{})
}
