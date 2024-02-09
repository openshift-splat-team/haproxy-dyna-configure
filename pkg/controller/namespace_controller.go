/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
	"github.com/openshift-splat-team/haproxy-dyna-configure/pkg"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// PodReconciler reconciles a HaproxyMetal object
type PodReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Context *ControllerContext
}

// NamespaceReconciler reconciles a HaproxyMetal object
type NamespaceReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Config  *data.MonitorConfig
	Context *ControllerContext
}

// +kubebuilder:rbac:groups=v1,resources=pods,verbs=get;list;watch;create;update;patch;delete
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	pod := &corev1.Pod{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, pod)
	if err != nil {
		return ctrl.Result{}, nil
	}

	r.Context.Update(pod)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Pod{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=v1,resources=namespaces,verbs=get;list;watch;create;update;patch;delete
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	ns := &corev1.Namespace{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, ns)

	if err != nil {
		r.Context.Destroy(req.Namespace)
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}

func StartManager() {
	var metricsAddr string
	var namespace string
	var enableLeaderElection bool
	var probeAddr string
	controllerContext := &ControllerContext{}

	config, err := pkg.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to get monitor config")
		os.Exit(1)
	}

	// Define the interval
	interval := 30 * time.Second

	// Create a new ticker
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Create a channel to signal the end of the program (optional)
	done := make(chan bool)

	// Start a goroutine that executes periodically
	go func() {
		for {
			select {
			case <-ticker.C:
				controllerContext.Reconcile()
			case <-done:
				return
			}
		}
	}()

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&namespace, "namespace", "vsphere-infra-helpers", "The namespace where ")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "0210028e.vanderlab.net",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	client := mgr.GetClient()
	corev1.AddToScheme(mgr.GetScheme())
	appsv1.AddToScheme(mgr.GetScheme())
	if err = (&NamespaceReconciler{
		Client:  client,
		Scheme:  mgr.GetScheme(),
		Config:  config,
		Context: controllerContext,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "namespace")
		os.Exit(1)
	}

	if err = (&PodReconciler{
		Client:  client,
		Scheme:  mgr.GetScheme(),
		Context: controllerContext,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "pod")
		os.Exit(1)
	}

	controllerContext.Initialize(config, client, namespace)

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}

}
