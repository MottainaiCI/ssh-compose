/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.

Based on the lxd-compose code
*/
package template_test

import (
	specs "github.com/MottainaiCI/ssh-compose/pkg/specs"
	. "github.com/MottainaiCI/ssh-compose/pkg/template"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("", func() {

	Context("TemplateJinja1", func() {

		proj := &specs.LxdCProject{
			Name: "project1",
			Environments: []specs.LxdCEnvVars{
				{
					EnvVars: map[string]interface{}{
						"key1": "value1",
						"key2": "value2",
						"key3": map[string]string{
							"f1": "foo",
							"f2": "foo2",
						},
					},
				},
			},
		}

		c := NewJinja2Compiler(proj)
		c.InitVars()

		It("Compilation1", func() {

			sourceData := `
k1: "{{ key1 }}"
k2: "{{ key2 }}"
`
			out, err := c.CompileRaw(sourceData)

			expectedOutput := `
k1: "value1"
k2: "value2"
`
			Expect(err).Should(BeNil())
			Expect(out).To(Equal(expectedOutput))
		})

	})
})
