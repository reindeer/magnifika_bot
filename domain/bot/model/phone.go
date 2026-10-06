package model

import "regexp"

var phoneRe = regexp.MustCompile(`^\+\d{11}$`)

func IsPhone(text string) bool {
	return phoneRe.MatchString(text)
}
