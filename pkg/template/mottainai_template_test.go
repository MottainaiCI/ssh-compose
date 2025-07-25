/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.

Based on the lxd-compose code
*/
package template_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	. "github.com/MottainaiCI/ssh-compose/pkg/template"
)

var _ = Describe("Template", func() {

	Describe("Draw", func() {
		Context("Using a simple template", func() {
			It("renders a specfile", func() {
				raw := `{{.EmailFrom}}`
				t := NewTemplate()
				t.Values["EmailFrom"] = "test"
				res, err := t.Draw(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(res).To(Equal("test"))
			})
		})
	})

	Describe("LoadValues", func() {
		Context("Using a simple template", func() {
			It("renders a specfile", func() {
				raw := `
values:
  image: 1

`
				t := NewTemplate()
				err := t.LoadValues(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(t.Values["image"]).To(Equal(1))
			})
		})
	})

	Describe("LoadArray", func() {
		Context("Load array values", func() {
			It("renders a specfile", func() {
				raw := `
values:
  images:
    - "image1"
    - "image2"

`

				t := NewTemplate()
				err := t.LoadValues(raw)
				Expect(err).ToNot(HaveOccurred())
				Expect(t.Values["images"]).To(Equal([]interface{}{"image1", "image2"}))
			})
		})
	})

})
