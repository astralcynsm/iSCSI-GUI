package iscsi

import "regexp"

var iqnPattern = regexp.MustCompile(`^iqn\.[0-9]{4}-[0-9]{2}\.[A-Za-z0-9.-]+:[A-Za-z0-9:._-]+$`)

func ValidIQN(iqn string) bool {
	return iqnPattern.MatchString(iqn)
}
