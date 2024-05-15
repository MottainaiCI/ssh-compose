/*
Copyright © 2024 Daniele Rondina <geaaru@macaronios.org>
See AUTHORS and LICENSE for the license details and contributors.
*/
package helpers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/MottainaiCI/ssh-compose/pkg/logger"
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

func Ask(msg string) bool {
	var input string

	log := logger.GetDefaultLogger()

	log.Msg("info", false, false, msg)
	_, err := fmt.Scanln(&input)
	if err != nil {
		return false
	}
	input = strings.ToLower(input)

	if input == "y" || input == "yes" {
		return true
	}

	return false
}
