package util

import (
	"fmt"
	"net"
	"strings"
)

func ResolveHost(hostname string) ([]net.IP, error) {
	if strings.Contains(hostname, "*") {
		hostname = fmt.Sprintf("test%s", hostname[1:])
	}
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %v", hostname, err)
	}
	return ips, nil
}
