package model

import (
	"regexp"
	"strings"
)

var plateRe = regexp.MustCompile(`^\s*[а-яА-Яa-zA-Z]\d{3}[а-яА-Яa-zA-Z]{2}\d{2,3}[\s.,]*$`)

func ParsePlate(text string) (string, bool) {
	if !plateRe.MatchString(text) {
		return "", false
	}
	return strings.Trim(text, ",. "), true
}
