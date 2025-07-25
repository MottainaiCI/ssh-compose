/*
Copyright © 2024-2025 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package specs

const (
	IncludeTypeAppend  = "append"
	IncludeTypePrepend = "prepend"
)

func (i *SshCInclude) GetType() string {
	ans := i.Type
	if ans == "" {
		ans = IncludeTypeAppend
	}
	return ans
}
func (i *SshCInclude) GetFiles() []string { return i.Files }
func (i *SshCInclude) IncludeInAppend() bool {
	if i.GetType() == IncludeTypeAppend {
		return true
	}
	return false
}
