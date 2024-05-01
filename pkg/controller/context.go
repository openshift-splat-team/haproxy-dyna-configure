package controllers

import (
	"context"
	"fmt"
	"log"
	"net/netip"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
	"github.com/openshift-splat-team/haproxy-dyna-configure/pkg"
	"github.com/openshift-splat-team/haproxy-dyna-configure/pkg/util"
	"github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	targetsMutex sync.Mutex
)

type ControllerContext struct {
	namespaceTargets  map[string]map[string]*data.NamespaceTarget
	config            *data.MonitorConfig
	client            client.Client
	namespace         string
	log               logr.Logger
	lastMonitorConfig *data.MonitorConfig
}

func getEnvFromPod(pod *corev1.Pod, varName string) (string, error) {
	for _, container := range pod.Spec.Containers {
		envVars := container.Env
		for _, envVar := range envVars {
			if envVar.Name == varName {
				return envVar.Value, nil
			}
		}
	}

	return "", fmt.Errorf("unable to find envvar %s", varName)

}
func (c *ControllerContext) Initialize(config *data.MonitorConfig, client client.Client, namespace string) {
	c.namespaceTargets = map[string]map[string]*data.NamespaceTarget{}
	c.config = config
	c.namespace = namespace
	c.log = zap.New()
	c.client = client
}

func (c *ControllerContext) Update(pod *corev1.Pod) {
	ns := pod.Namespace
	if !strings.HasPrefix(ns, "ci-ln-") {
		return
	}
	var jobHash string
	jobHash, err := getEnvFromPod(pod, "JOB_NAME_HASH")
	if err != nil {
		log.Printf("no job hash is associated with this pod: %v", err)
		return
	}

	targetsMutex.Lock()
	defer targetsMutex.Unlock()

	var exists bool
	var namespaceTarget map[string]*data.NamespaceTarget
	if namespaceTarget, exists = c.namespaceTargets[ns]; !exists {
		namespaceTarget = map[string]*data.NamespaceTarget{}
	}

	if _, exists := namespaceTarget[jobHash]; !exists {
		namespaceTarget[jobHash] = &data.NamespaceTarget{
			Namespace: ns,
			JobHash:   jobHash,
		}
	}
	c.namespaceTargets[ns] = namespaceTarget
}

func (c *ControllerContext) getBaseDomain(ns, jobHash string) string {
	return fmt.Sprintf("%s-%s.%s", ns, jobHash, c.config.BaseDomain)
}

func (c *ControllerContext) reconcileTargets() *data.MonitorConfig {
	monitorConfig := data.MonitorConfig{
		BaseDomain:    c.config.BaseDomain,
		MonitorRanges: []data.MonitorRange{},
		HaproxyHeader: c.config.HaproxyHeader,
	}

	logrus.Infof("number of namespaces: %d", len(c.namespaceTargets))
	for ns, jobs := range c.namespaceTargets {
		for jobHash, job := range jobs {
			ports := []data.MonitorPort{}

			if len(job.APIVIP) > 0 {
				ports = append(ports,
					data.MonitorPort{
						Port:      6443,
						PathMatch: "api.",
						Targets:   job.APIVIP,
					})
			}
			if len(job.IngressVIP) > 0 {
				ports = append(ports,
					data.MonitorPort{
						Port:       443,
						PathPrefix: "*.apps.",
						Targets:    job.IngressVIP,
					})
			}
			if len(ports) == 0 {
				continue
			}
			monitorConfig.MonitorRanges = append(monitorConfig.MonitorRanges, data.MonitorRange{
				BaseDomain:   c.getBaseDomain(ns, jobHash),
				MonitorPorts: ports,
			})
		}
	}
	return &monitorConfig
}

func (c *ControllerContext) hasConfigUpdated(monitorConfig *data.MonitorConfig) bool {
	defer func() {
		c.lastMonitorConfig = monitorConfig
	}()

	if c.lastMonitorConfig == nil {
		return true
	}

	monitorRangeMap := map[string]data.MonitorRange{}
	prevMonitorRangeMap := map[string]data.MonitorRange{}
	for _, monitorRange := range monitorConfig.MonitorRanges {
		monitorRangeMap[monitorRange.BaseDomain] = monitorRange
	}

	for _, monitorRange := range c.lastMonitorConfig.MonitorRanges {
		prevMonitorRangeMap[monitorRange.BaseDomain] = monitorRange
		if _, exists := monitorRangeMap[monitorRange.BaseDomain]; !exists {
			fmt.Printf("mismatch: base domain %s does not exist", monitorRange.BaseDomain)
			return true
		}
	}

	for _, monitorRange := range monitorConfig.MonitorRanges {
		if _, exists := prevMonitorRangeMap[monitorRange.BaseDomain]; !exists {
			fmt.Printf("mismatch: base domain %s does not exist", monitorRange.BaseDomain)
			return true
		}
	}

	return false
}

func (c *ControllerContext) Reconcile() {
	logrus.Infof("reconciling HAProxy configuration")
	ctx := context.TODO()
	if err := c.CheckForARecords(); err != nil {
		log.Printf("error while checking A records: %v", err)
		return
	}

	targetsMutex.Lock()
	defer targetsMutex.Unlock()

	monitorConfig := c.reconcileTargets()

	if !c.hasConfigUpdated(monitorConfig) {
		return
	}

	logrus.Infof("configuration has updated, building new haproxy configuration")

	content, hash, err := pkg.BuildTargetHAProxyConfig(monitorConfig)
	if err != nil {
		log.Printf("unable to build HAProxy config: %v", err)
		return
	}

	cm := corev1.ConfigMap{}

	cmName := types.NamespacedName{
		Namespace: c.namespace,
		Name:      "haproxy",
	}

	create := false
	if err = c.client.Get(ctx, cmName, &cm); err != nil {
		create = true
	}

	cm = corev1.ConfigMap{
		ObjectMeta: v1.ObjectMeta{
			Name:      "haproxy",
			Namespace: c.namespace,
			Annotations: map[string]string{
				"config-hash": hash,
			},
		},
		Data: map[string]string{
			"haproxy.cfg": content,
		},
	}

	if create {
		if err = c.client.Create(ctx, &cm); err != nil {
			c.log.V(4).Info("creating haproxy configmap")
			log.Printf("unable to create config map: %v", err)
			return
		}
	} else {
		if err = c.client.Update(ctx, &cm); err != nil {
			log.Printf("error updating haproxy configmap: %v", err)
			return
		}
	}

	c.bumpHaproxyDeployment(ctx, hash)
}

func (c *ControllerContext) bumpHaproxyDeployment(ctx context.Context, hash string) {
	deploymentName := types.NamespacedName{
		Namespace: c.namespace,
		Name:      "clusterbot-haproxy",
	}

	deployment := appsv1.Deployment{}

	create := false
	if err := c.client.Get(ctx, deploymentName, &deployment); err != nil {
		create = true
	}

	if create {
		logrus.Infof("creating haproxy configmap")
		if err := c.client.Create(ctx, &deployment); err != nil {
			log.Printf("unable to create config map: %v", err)
			return
		}
	} else {
		logrus.Infof("updating haproxy configmap")
		if deployment.Spec.Template.Annotations == nil {
			deployment.Spec.Template.Annotations = map[string]string{}
		}
		deployment.Spec.Template.Annotations["config-hash"] = hash
		if err := c.client.Update(ctx, &deployment); err != nil {
			log.Printf("unable to update config map: %v", err)
			return
		}
	}
}

func (c *ControllerContext) CheckForARecords() error {
	hostsToCheck := map[string]string{}
	targetHostCheckMap := map[string]*data.NamespaceTarget{}

	targetsMutex.Lock()
	for ns, namespaceTarget := range c.namespaceTargets {
		for hashId, target := range namespaceTarget {
			if len(target.APIVIP) == 0 {
				url := fmt.Sprintf("api-int.%s-%s.%s", ns, hashId, c.config.BaseDomain)
				hostsToCheck[url] = ""
				targetHostCheckMap[url] = target
			}
		}
	}
	targetsMutex.Unlock()

	for host := range hostsToCheck {
		ips, err := util.ResolveHost(host)
		if err != nil {
			// unable to resolve host, continue
			fmt.Printf("unable to resolve %s: %v\n", host, err)
			continue
		}
		for _, ip := range ips {
			ipStr := ip.String()
			if !strings.HasPrefix(ipStr, "10.") {
				continue
			}
			hostsToCheck[host] = ipStr
		}
	}

	targetsMutex.Lock()
	defer targetsMutex.Unlock()

	for host, ip := range hostsToCheck {
		if len(ip) == 0 {
			continue
		}
		var target *data.NamespaceTarget
		var exists bool
		if target, exists = targetHostCheckMap[host]; !exists {
			continue
		}
		target.APIVIP = []string{ip}
		target.IngressVIP = []string{ip}

		ip, err := netip.ParseAddr(ip) // Parse IP address
		if err != nil {
			fmt.Println("Invalid IP address")
		} else {
			fmt.Println("Original IP address:", ip.String())
			newIp := ip.Next() // get the next address
			fmt.Println("Incremented IP address:", newIp.String())
			target.IngressVIP = append(target.IngressVIP, newIp.String())
		}
		c.namespaceTargets[target.Namespace][target.JobHash] = target
	}
	return nil
}

func (c *ControllerContext) Destroy(namespace string) {
	logrus.Infof("destroying namespace %s", namespace)
	targetsMutex.Lock()
	defer targetsMutex.Unlock()
	if _, exists := c.namespaceTargets[namespace]; exists {
		logrus.Infof("deleting namespace %s", namespace)
		delete(c.namespaceTargets, namespace)
	}
}
