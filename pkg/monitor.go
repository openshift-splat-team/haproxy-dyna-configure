package pkg

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-yaml/yaml"
	"github.com/netdata/go.d.plugin/pkg/iprange"
	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var monitorConfig data.MonitorConfigSpec
var mu sync.Mutex

const (
	defaultConfigPath   = "monitor-config.yaml"
	monitorConfigEnvVar = "MONITOR_CONFIG"
)

func getConfigPath() string {
	if val, exists := os.LookupEnv(monitorConfigEnvVar); exists {
		return val
	}
	return defaultConfigPath
}

func GetConfig() (*data.MonitorConfig, error) {
	var monitorConfig data.MonitorConfig
	configRaw, err := os.ReadFile(getConfigPath())
	if err != nil {
		return nil, fmt.Errorf("unable to read monitor config: %v", err)
	}
	err = yaml.Unmarshal(configRaw, &monitorConfig)
	if len(monitorConfig.BaseDomain) == 0 {
		monitorConfig.BaseDomain = "vmc-ci.devcluster.openshift.com"
	}
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall monitor config: %v", err)
	}
	return &monitorConfig, nil
}

func Initialize(ctx context.Context) error {

	monitorConfig, err := GetConfig()
	if err != nil {
		return fmt.Errorf("unable to get config: %v", err)
	}

	if len(monitorConfig.SubnetsJson) > 0 {
		nativeSubnetRanges, err := parseSubnetsJson(monitorConfig.SubnetsJson)
		if err != nil {
			return errors.Wrap(err, "unable to parse native subnet json")
		}
		monitorConfig.MonitorRanges = append(monitorConfig.MonitorRanges, nativeSubnetRanges...)
	}
	return nil
}

func CheckRanges(ctx context.Context) (*data.MonitorConfigSpec, error) {
	var wg sync.WaitGroup
	const maxThreads = 10
	var activeThreads = 0

	for idx := range monitorConfig.MonitorConfig.MonitorRanges {
		if activeThreads >= maxThreads {
			wg.Wait()
			activeThreads = 0
		}
		wg.Add(1)
		activeThreads++
		go CheckRange(ctx, &wg, &monitorConfig.MonitorConfig.MonitorRanges[idx])
	}
	wg.Wait()
	return &monitorConfig, nil
}

func CheckPort(ctx context.Context, wg *sync.WaitGroup, monitorPort *data.MonitorPort, monitorRange *data.MonitorRange, ip string) {
	defer wg.Done()
	client := http.Client{
		Timeout: time.Duration(monitorConfig.MonitorConfig.CheckTimeout) * time.Millisecond,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	protocol := monitorPort.Protocol
	if len(protocol) == 0 {
		protocol = "https"
		mu.Lock()
		monitorPort.Protocol = protocol
		mu.Unlock()
	}
	url := fmt.Sprintf("%s://%s:%d", protocol, ip, monitorPort.Port)
	logrus.Debugf("checking URL %s", url)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		logrus.Error(err)
		return
	}
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	mu.Lock()
	monitorPort.Targets = append(monitorPort.Targets, ip)
	mu.Unlock()
	if resp.TLS != nil {
		for _, cert := range resp.TLS.PeerCertificates {
			for _, dnsname := range cert.DNSNames {
				var prefix string
				mu.Lock()
				if len(monitorPort.PathPrefix) > 0 {
					prefix = monitorPort.PathPrefix
				} else if len(monitorPort.PathMatch) > 0 {
					prefix = monitorPort.PathMatch
				}
				mu.Unlock()
				if strings.HasPrefix(dnsname, prefix) {
					splits := strings.SplitAfter(dnsname, prefix)
					if len(splits) < 2 {
						continue
					}

					mu.Lock()
					monitorRange.BaseDomain = splits[1]
					logrus.Infof("found base domain %s", monitorRange.BaseDomain)
					mu.Unlock()
				}
			}
		}
	}

}

func CheckRange(ctx context.Context, cWaitGroup *sync.WaitGroup, monitorRange *data.MonitorRange) {
	defer cWaitGroup.Done()
	parseRange, err := iprange.ParseRange(fmt.Sprintf("%s-%s", monitorRange.IpAddressStart, monitorRange.IpAddressEnd))
	if err != nil {
		logrus.Error(err)
		return
	}

	ip, err := netip.ParseAddr(monitorRange.IpAddressStart)
	if err != nil {
		logrus.Error(err)
		return
	}

	for idx := range monitorRange.MonitorPorts {
		monitorRange.MonitorPorts[idx].Targets = []string{}
	}
	var wg sync.WaitGroup
	const maxThreads = 25
	var activeThreads = 0
	for parseRange.Contains(net.ParseIP(ip.String())) {
		for idx := range monitorRange.MonitorPorts {
			if activeThreads >= maxThreads {
				wg.Wait()
				activeThreads = 0
			}
			wg.Add(1)
			activeThreads++
			go CheckPort(ctx, &wg, &monitorRange.MonitorPorts[idx], monitorRange, ip.String())
		}
		ip = ip.Next()
	}
	wg.Wait()
}
