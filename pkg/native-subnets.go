package pkg

import (
	"encoding/json"
	"os"

	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func parseSubnetsJson(subnetsFile string) ([]data.MonitorRange, error) {
	logrus.Infof("reading subnets from %s", subnetsFile)
	subnetsBytes, err := os.ReadFile(subnetsFile)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read %s", subnetsFile)
	}

	var subnetsUntyped map[string]interface{}
	// Parse the JSON data into the map
	err = json.Unmarshal([]byte(subnetsBytes), &subnetsUntyped)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to parse")
	}

	monitorRanges := []data.MonitorRange{}
	for datacenter, vlans := range subnetsUntyped {
		logrus.Infof("traversing datacenter %s", datacenter)
		for vlan, subnetUntyped := range vlans.(map[string]interface{}) {
			logrus.Infof("traversing vlan %s", vlan)
			subnet := subnetUntyped.(map[string]interface{})
			ipAddresses := subnet["ipAddresses"].([]interface{})

			monitorRange := data.MonitorRange{
				IpAddressStart: ipAddresses[0].(string),
				IpAddressEnd:   ipAddresses[10].(string),
				MonitorPorts: []data.MonitorPort{
					{
						Port:      6443,
						Name:      "api",
						PathMatch: "api",
					},
					{
						Port:       443,
						Name:       "ingress-https",
						PathPrefix: "*.apps",
					},
				},
			}
			monitorRanges = append(monitorRanges, monitorRange)
		}
	}
	return monitorRanges, nil
}
