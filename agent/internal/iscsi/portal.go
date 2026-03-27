package iscsi

import (
	"net"
	"strings"
)

func ValidPortalIP(ip string) bool {
	ip = strings.TrimSpace(ip)
	return net.ParseIP(ip) != nil
}

func ValidPortalPort(port int) bool {
	return port >= 1 && port <= 65535
}
