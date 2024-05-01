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
	fmt.Printf("checking host %s\n", hostname)
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve %s: %v", hostname, err)
	}
	fmt.Printf("IPs: %v", ips)
	return ips, nil
}
