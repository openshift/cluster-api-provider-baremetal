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
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	bmoapis "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	capm3apis "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	osconfigv1 "github.com/openshift/api/config/v1"
	apifeatures "github.com/openshift/api/features"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	utiltls "github.com/openshift/controller-runtime-common/pkg/tls"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/apis"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/baremetal"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/cloud/baremetal/actuators/machine"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/controller"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/controller/metal3remediation"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/manager/wrapper"
	capbmwebhook "github.com/openshift/cluster-api-provider-baremetal/pkg/webhook"
	"github.com/openshift/library-go/pkg/features"
	maomachine "github.com/openshift/machine-api-operator/pkg/controller/machine"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/util/feature"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/component-base/featuregate"
	"k8s.io/klog/v2"
	"k8s.io/klog/v2/klogr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

const (
	// Align with MachineWebhookPort in https://github.com/openshift/machine-api-operator/blob/master/pkg/operator/sync.go
	// or import from there when relevant change was merged
	defaultWebhookPort    = 8440
	defaultWebhookCertdir = "/etc/machine-api-operator/tls"
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

	webhookEnabled := flag.Bool("webhook-enabled", true,
		"Webhook server, enabled by default. When enabled, the manager will run a webhook server.")

	webhookPort := flag.Int("webhook-port", defaultWebhookPort,
		"Webhook Server port, only used when webhook-enabled is true.")

	webhookCertdir := flag.String("webhook-cert-dir", defaultWebhookCertdir,
		"Webhook cert dir, only used when webhook-enabled is true.")

	tlsCipherSuites := flag.String("tls-cipher-suites", "",
		"Comma-separated list of TLS cipher suites.")

	tlsMinVersion := flag.String("tls-min-version", "",
		"Minimum TLS version supported.")

	// Sets up feature gates
	defaultMutableGate := feature.DefaultMutableFeatureGate
	gateOpts, err := features.NewFeatureGateOptions(defaultMutableGate, apifeatures.SelfManaged, apifeatures.FeatureGateMachineAPIMigration)
	if err != nil {
		klog.Fatalf("Error setting up feature gates: %v", err)
	}

	// Add the --feature-gates flag
	gateOpts.AddFlagsToGoFlagSet(nil)

	flag.Parse()

	log := logf.Log.WithName("baremetal-controller-manager")
	logf.SetLogger(klogr.New())
	entryLog := log.WithName("entrypoint")

	cfg := config.GetConfigOrDie()
	if cfg == nil {
		panic(fmt.Errorf("GetConfigOrDie didn't die"))
	}

	err = waitForAPIs(cfg)
	if err != nil {
		entryLog.Error(err, "unable to discover required APIs")
		os.Exit(1)
	}

	var watchNamespaces map[string]cache.Config
	if *watchNamespace != "" {
		watchNamespaces = map[string]cache.Config{
			*watchNamespace: {},
		}
		klog.Infof("Watching machine-api objects only in namespace %q for reconciliation.", *watchNamespace)
	}

	// Sets feature gates from flags
	klog.Infof("Initializing feature gates: %s", strings.Join(defaultMutableGate.KnownFeatures(), ", "))
	warnings, err := gateOpts.ApplyTo(defaultMutableGate)
	if err != nil {
		klog.Fatalf("Error setting feature gates from flags: %v", err)
	}
	if len(warnings) > 0 {
		klog.Infof("Warnings setting feature gates from flags: %v", warnings)
	}

	klog.Infof("FeatureGateMachineAPIMigration initialised: %t", defaultMutableGate.Enabled(featuregate.Feature(apifeatures.FeatureGateMachineAPIMigration)))

	// Setup a Manager
	opts := manager.Options{
		HealthProbeBindAddress:  *healthAddr,
		Metrics:                 metricsserver.Options{BindAddress: *metricsAddr},
		LeaderElection:          *leaderElect,
		LeaderElectionID:        "controller-leader-election-capbm",
		LeaderElectionNamespace: *leaderElectResourceNamespace,
		LeaseDuration:           leaderElectLeaseDuration,
		// Slow the default retry and renew election rate to reduce etcd writes at idle: BZ 1858400
		RetryPeriod:   &retryPeriod,
		RenewDeadline: &renewDeadline,
		Cache: cache.Options{
			DefaultNamespaces: watchNamespaces,
		},
	}

	if *webhookEnabled {
		tlsProfile := osconfigv1.TLSProfileSpec{
			MinTLSVersion: osconfigv1.TLSProtocolVersion(*tlsMinVersion),
		}
		if *tlsCipherSuites != "" {
			tlsProfile.Ciphers = strings.Split(*tlsCipherSuites, ",")
		}

		tlsOpts, _ := utiltls.NewTLSConfigFromProfile(tlsProfile)

		opts.WebhookServer = webhook.NewServer(webhook.Options{
			Port:    *webhookPort,
			CertDir: *webhookCertdir,
			TLSOpts: []func(*tls.Config){tlsOpts},
		})
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

	if err := capm3apis.AddToScheme(mgr.GetScheme()); err != nil {
		panic(err)
	}

	// the manager wrapper will add an extra Watch to the controller
	maomachine.AddWithActuator(wrapper.New(mgr), machineActuator, defaultMutableGate)

	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "Failed to add controller to manager")
		os.Exit(1)
	}

	// Set up the context that's going to be used in controllers and for the manager.
	ctx := signals.SetupSignalHandler()

	if err := (&metal3remediation.Metal3RemediationReconciler{
		Client:         mgr.GetClient(),
		ManagerFactory: baremetal.NewManagerFactory(mgr.GetClient()),
		Log:            ctrl.Log.WithName("controllers").WithName("Metal3Remediation"),
	}).SetupWithManager(ctx, mgr); err != nil {
		log.Error(err, "unable to create controller", "controller", "Metal3Remediation")
		os.Exit(1)
	}

	if *webhookEnabled {
		if err := (&capbmwebhook.Metal3Remediation{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "Metal3Remediation")
			os.Exit(1)
		}

		if err := (&capbmwebhook.Metal3RemediationTemplate{}).SetupWebhookWithManager(mgr); err != nil {
			log.Error(err, "unable to create webhook", "webhook", "Metal3RemediationTemplate")
			os.Exit(1)
		}
	}

	if err := mgr.AddReadyzCheck("ping", healthz.Ping); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.AddHealthzCheck("ping", healthz.Ping); err != nil {
		klog.Fatal(err)
	}

	if err := mgr.Start(ctx); err != nil {
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
