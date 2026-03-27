package iscsi

import "regexp"

var (
	backstoreNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,64}$`)
	backstorePathPattern = regexp.MustCompile(`^/[^\s]+$`)
)

func ValidBackstoreType(t string) bool {
	return t == "fileio" || t == "block"
}

func ValidBackstoreName(name string) bool {
	return backstoreNamePattern.MatchString(name)
}

func ValidBackstorePath(path string) bool {
	return backstorePathPattern.MatchString(path)
}
