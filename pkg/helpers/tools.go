/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"regexp"
)

func RegexEntry(regexString string, listEntries []string) []string {
	ans := []string{}

	r := regexp.MustCompile(regexString)
	for _, e := range listEntries {

		if r != nil && r.MatchString(e) {
			ans = append(ans, e)
		}
	}
	return ans
}
