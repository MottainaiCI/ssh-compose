/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.

Based on the lxd-compose code
*/
package template

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Values map[string]interface{}
}

func NewTemplate() *Template { return &Template{Values: map[string]interface{}{}} }

func (tem *Template) DrawFromFile(file string) (string, error) {
	dat, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	return tem.Draw(string(dat))
}
func (tem *Template) AppendValue(k string, v interface{}) {
	if _, ok := tem.Values[k]; !ok {
		tem.Values[k] = v
	}
}
func (tem *Template) LoadValuesFromFile(file string) error {
	dat, err := os.ReadFile(file)
	if err != nil {
		return err
	}
	return tem.LoadValues(string(dat))
}
func (tem *Template) LoadValues(raw string) error {

	m, err := tem.ReadValues(raw)
	if err != nil {
		return err
	}
	vals, ok := m["values"]

	if !ok {
		return errors.New("No values defined in the values: section")
	}

	for k, v := range vals {

		tem.AppendValue(k, v)

	}

	return nil
}

func (tem *Template) ReadValues(raw string) (map[string]map[string]interface{}, error) {
	m := make(map[string]map[string]interface{})

	err := yaml.Unmarshal([]byte(raw), &m)
	if err != nil {
		return m, err
	}

	return m, nil
}

func (tem *Template) Draw(raw string) (string, error) {
	tf := sprig.TxtFuncMap()
	tf["isInt"] = func(i interface{}) bool {
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.Int, reflect.Int8, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
			return true
		default:
			return false
		}
	}
	tf["isString"] = func(i interface{}) bool {
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.String:
			return true
		default:
			return false
		}
	}
	tf["isSlice"] = func(i interface{}) bool {
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.Slice:
			return true
		default:
			return false
		}
	}
	tf["isArray"] = func(i interface{}) bool {
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.Array:
			return true
		default:
			return false
		}
	}
	tf["isMap"] = func(i interface{}) bool {
		v := reflect.ValueOf(i)
		switch v.Kind() {
		case reflect.Map:
			return true
		default:
			return false
		}
	}
	tf["replaceAll"] = strings.ReplaceAll
	tf["join"] = strings.Join
	tf["joinWithPrefix"] = func(a []string, sep, prefix string) string {
		var ans []string
		for _, elem := range a {
			ans = append(ans, fmt.Sprintf("%s%s", prefix, elem))
		}
		return strings.Join(ans, sep)
	}
	tf["joinWithPrefixAndPostfix"] = func(a []string, sep, prefix, postfix string) string {
		var ans []string
		for _, elem := range a {
			ans = append(ans, fmt.Sprintf("%s%s%s", prefix, elem, postfix))
		}
		return strings.Join(ans, sep)
	}
	tf["sort"] = func(a []string) []string {
		sort.Strings(a)
		return a
	}
	tf["getKeys"] = func(m map[interface{}]interface{}) []string {
		var ans []string
		for k := range m {
			ans = append(ans, k.(string))
		}
		return ans
	}
	tf["cast2StringArray"] = func(a []interface{}) []string {
		var ans []string
		for _, v := range a {
			ans = append(ans, v.(string))
		}
		return ans
	}
	t := template.New("spec").Funcs(tf)
	tt, err := t.Parse(raw)
	if err != nil {
		return "", err
	}
	var doc bytes.Buffer
	if err = tt.Execute(&doc, &tem.Values); err != nil {
		return "", err
	}
	return doc.String(), nil
}
