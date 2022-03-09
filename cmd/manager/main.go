/*
Copyright 2019 The Kubernetes Authors.

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

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	bmoapis "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	"github.com/metal3-io/cluster-api-provider-metal3/pkg/apis"
	"github.com/metal3-io/cluster-api-provider-metal3/pkg/cloud/baremetal/actuators/machine"
	"github.com/metal3-io/cluster-api-provider-metal3/pkg/controller"
	"github.com/metal3-io/cluster-api-provider-metal3/pkg/manager/wrapper"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	maomachine "github.com/openshift/machine-api-operator/pkg/controller/machine"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

// The default durations for the leader election operations.
var (
	leaseDuration = 120 * time.Second
	renewDeadline = 110 * time.Second
	retryPeriod   = 20 * time.Second
)

func main() {
	klog.InitFlags(nil)

	watchNamespace := flag.String(
		"namespace",
		"",
		"Namespace that the controller watches to reconcile machine-api objects. If unspecified, the controller watches for machine-api objects across all namespaces.",
	)

	healthAddr := flag.String(
		"health-addr",
		":9440",
		"The address for health checking.",
	)

	// Default machine metrics address defined by MAO - https://github.com/openshift/machine-api-operator/blob/master/pkg/metrics/metrics.go#L16
	metricsAddr := flag.String(
		"metrics-addr",
		":8081",
		"The address the metric endpoint binds to.",
	)

	leaderElectResourceNamespace := flag.String(
		"leader-elect-resource-namespace",
		"",
		"The namespace of resource object that is used for locking during leader election. If unspecified and running in cluster, defaults to the service account namespace for the controller. Required for leader-election outside of a cluster.",
	)

	leaderElect := flag.Bool(
		"leader-elect",
		false,
		"Start a leader election client and gain leadership before executing the main loop. Enable this when running replicated components for high availability. This will ensure only one of the old or new controller is running at a time, allowing safe upgrades and recovery.",
	)

	leaderElectLeaseDuration := flag.Duration(
		"leader-elect-lease-duration",
		leaseDuration,
		"The duration that non-leader candidates will wait after observing a leadership renewal until attempting to acquire leadership of a led but unrenewed leader slot. This is effectively the maximum duration that a leader can be stopped before it is replaced by another candidate. This is only applicable if leader election is enabled.",
	)
	flag.Parse()

	log := logf.Log.WithName("baremetal-controller-manager")
	logf.SetLogger(klogr.New())
	entryLog := log.WithName("entrypoint")

	cfg := config.GetConfigOrDie()
	if cfg == nil {
		panic(fmt.Errorf("GetConfigOrDie didn't die"))
	}

	err := waitForAPIs(cfg)
	if err != nil {
		entryLog.Error(err, "unable to discover required APIs")
		os.Exit(1)
	}

	// Setup a Manager
	opts := manager.Options{
		HealthProbeBindAddress:  *healthAddr,
		MetricsBindAddress:      *metricsAddr,
		LeaderElection:          *leaderElect,
		LeaderElectionID:        "controller-leader-election-capbm",
		LeaderElectionNamespace: *leaderElectResourceNamespace,
		LeaseDuration:           leaderElectLeaseDuration,
		// Slow the default retry and renew election rate to reduce etcd writes at idle: BZ 1858400
		RetryPeriod:   &retryPeriod,
		RenewDeadline: &renewDeadline,
	}
	if *watchNamespace != "" {
		opts.Namespace = *watchNamespace
		klog.Infof("Watching machine-api objects only in namespace %q for reconciliation.", opts.Namespace)
	}

	mgr, err := manager.New(cfg, opts)
	if err != nil {
		entryLog.Error(err, "unable to set up overall controller manager")
		os.Exit(1)
	}

	machineActuator, err := machine.NewActuator(machine.ActuatorParams{
		Client: mgr.GetClient(),
	})
	if err != nil {
		panic(err)
	}

	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	if err := machinev1beta1.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	if err := bmoapis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	// the manager wrapper will add an extra Watch to the controller
	maomachine.AddWithActuator(wrapper.New(mgr), machineActuator)

	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "Failed to add controller to manager")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		entryLog.Error(err, "unable to run manager")
		os.Exit(1)
	}
}

func waitForAPIs(cfg *rest.Config) error {
	log := logf.Log.WithName("baremetal-controller-manager")

	c, err := discovery.NewDiscoveryClientForConfig(cfg)
	if err != nil {
		return err
	}

	metal3GV := schema.GroupVersion{
		Group:   "metal3.io",
		Version: "v1alpha1",
	}

	for {
		err = discovery.ServerSupportsVersion(c, metal3GV)
		if err != nil {
			log.Info(fmt.Sprintf("Waiting for API group %v to be available: %v", metal3GV, err))
			time.Sleep(time.Second * 10)
			continue
		}
		log.Info(fmt.Sprintf("Found API group %v", metal3GV))
		break
	}

	return nil
}
