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

package machineset

import (
	"encoding/json"
	"testing"

	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	bmoapis "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	bmv1alpha1 "github.com/metal3-io/cluster-api-provider-metal3/pkg/apis/baremetal/v1alpha1"
	machinev1beta1 "github.com/openshift/machine-api-operator/pkg/apis/machine/v1beta1"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var expectedRequest1 = reconcile.Request{NamespacedName: types.NamespacedName{Name: "machineset1", Namespace: "default"}}
var expectedRequest2 = reconcile.Request{NamespacedName: types.NamespacedName{Name: "machineset2", Namespace: "default"}}
var machinesetKey1 = types.NamespacedName{Name: "machineset1", Namespace: "default"}
var machinesetKey2 = types.NamespacedName{Name: "machineset2", Namespace: "default"}

func TestHostMatches(t *testing.T) {
	scheme := runtime.NewScheme()
	machinev1beta1.AddToScheme(scheme)
	ctx := context.TODO()

	testCases := []struct {
		Host          *bmh.BareMetalHost
		HSelector     labels.Selector
		MSSelector    labels.Selector
		Machines      []runtime.Object
		ExpectMatch   bool
		ExpectMessage string
	}{
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "large",
			}),
			MSSelector:    labels.NewSelector(),
			ExpectMatch:   true,
			ExpectMessage: "Expected match: available host has matching label",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "extralarge",
			}),
			MSSelector:    labels.NewSelector(),
			ExpectMatch:   false,
			ExpectMessage: "Expected no match: available host has non-matching label value",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
				Spec: bmh.BareMetalHostSpec{
					ConsumerRef: &v1.ObjectReference{
						Kind:       "Machine",
						APIVersion: machinev1beta1.SchemeGroupVersion.String(),
						Name:       "machine1",
						Namespace:  "default",
					},
				},
			},
			Machines: []runtime.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "machine1",
						Namespace: "default",
						Labels:    map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-storage"},
					},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "extralarge",
			}),
			MSSelector: labels.SelectorFromSet(map[string]string{
				"machine.openshift.io/cluster-api-machineset": "cluster0-storage",
			}),
			ExpectMatch:   true,
			ExpectMessage: "Expected match: host consumer is a Machine that matches the MSSelector, even though the host has a non-matching label value",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
				Spec: bmh.BareMetalHostSpec{
					ConsumerRef: &v1.ObjectReference{
						Kind:       "Machine",
						APIVersion: machinev1beta1.SchemeGroupVersion.String(),
						Name:       "machine1",
						Namespace:  "default",
					},
				},
			},
			Machines: []runtime.Object{
				&machinev1beta1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "machine1",
						Namespace: "default",
						Labels:    map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-workers"},
					},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "large",
			}),
			MSSelector: labels.SelectorFromSet(map[string]string{
				"machine.openshift.io/cluster-api-machineset": "cluster0-storage",
			}),
			ExpectMatch:   false,
			ExpectMessage: "Expected no match: host consumer is a Machine that does not match the MSSelector",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
				Spec: bmh.BareMetalHostSpec{
					ConsumerRef: &v1.ObjectReference{
						Kind:       "NotAMachine",
						APIVersion: machinev1beta1.SchemeGroupVersion.String(),
						Name:       "machine1",
						Namespace:  "default",
					},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "large",
			}),
			MSSelector:    labels.NewSelector(),
			ExpectMatch:   false,
			ExpectMessage: "Expected no match: host consumer is not a Machine",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
				Spec: bmh.BareMetalHostSpec{
					ConsumerRef: &v1.ObjectReference{
						Kind:       "Machine",
						APIVersion: "fail.metal3.io/v1alpha1",
						Name:       "machine1",
						Namespace:  "default",
					},
				},
			},
			HSelector: labels.SelectorFromSet(map[string]string{
				"size": "large",
			}),
			MSSelector:    labels.NewSelector(),
			ExpectMatch:   false,
			ExpectMessage: "Expected no match: host consumer is not a Machine at the right API group/version",
		},
	}

	for _, tc := range testCases {
		c := fakeclient.NewFakeClientWithScheme(scheme, tc.Machines...)
		reconciler := ReconcileMachineSet{
			Client: c,
			scheme: scheme,
		}
		result, err := reconciler.hostMatches(ctx, tc.HSelector, tc.MSSelector, tc.Host)
		if err != nil {
			t.Errorf("%v", err)
		}
		if result != tc.ExpectMatch {
			t.Logf(tc.ExpectMessage)
			t.FailNow()
		}
	}
}

func TestScale(t *testing.T) {
	scheme := runtime.NewScheme()
	machinev1beta1.AddToScheme(scheme)
	bmoapis.AddToScheme(scheme)

	rawProviderSpec, err := json.Marshal(&bmv1alpha1.BareMetalMachineProviderSpec{
		HostSelector: bmv1alpha1.HostSelector{
			MatchLabels: map[string]string{"size": "large"},
		},
	})
	if err != nil {
		t.Errorf("%v", err)
	}

	rep := int32(17)
	instance := &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        machinesetKey1.Name,
			Namespace:   machinesetKey1.Namespace,
			Annotations: map[string]string{AutoScaleAnnotation: "yesplease"},
		},
		Spec: machinev1beta1.MachineSetSpec{
			Template: machinev1beta1.MachineTemplateSpec{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{Raw: rawProviderSpec},
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-worker"},
			},
			Replicas: &rep,
		},
	}
	// machine1 has a different label than the MachineSet's Selector, so its
	// consumed host should not be counted as part of that MachineSet's
	// potential hosts.
	machine1 := machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "machine1",
			Namespace: "default",
			Labels:    map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-storage"},
		},
	}
	// machine2 has a label that matches the MachineSet's Selector, so its
	// consumed host should be counted as part of that MachineSet's potential
	// hosts even if that BareMetalHost does not otherwise match.
	machine2 := machinev1beta1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "machine2",
			Namespace: "default",
			Labels:    map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-worker"},
		},
	}
	host1 := bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host1",
			Namespace: "default",
			Labels:    map[string]string{"size": "large"},
		},
	}
	// This host has a different label, but its consuming Machine is
	// part of the MachineSet, so it should be counted.
	host2 := bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host2",
			Namespace: "default",
			Labels:    map[string]string{"size": "small"},
		},
		Spec: bmh.BareMetalHostSpec{
			ConsumerRef: &v1.ObjectReference{
				Kind:       "Machine",
				APIVersion: machinev1beta1.SchemeGroupVersion.String(),
				Name:       "machine2",
				Namespace:  "default",
			},
		},
	}
	// This host has a different label, so it should not match
	host3 := bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host3",
			Namespace: "default",
			Labels:    map[string]string{"size": "extramedium"},
		},
	}
	// This host is consumed by a Machine in a different MachineSet, so it
	// should not be counted
	host4 := bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host4",
			Namespace: "default",
			Labels:    map[string]string{"size": "large"},
		},
		Spec: bmh.BareMetalHostSpec{
			ConsumerRef: &v1.ObjectReference{
				Kind:       "Machine",
				APIVersion: machinev1beta1.SchemeGroupVersion.String(),
				Name:       "machine1",
				Namespace:  "default",
			},
		},
	}
	// This host is consumed by something that is not a Machine, so it should
	// not be counted
	host5 := bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host5",
			Namespace: "default",
			Labels:    map[string]string{"size": "large"},
		},
		Spec: bmh.BareMetalHostSpec{
			ConsumerRef: &v1.ObjectReference{
				Kind:       "NotAMachine",
				APIVersion: machinev1beta1.SchemeGroupVersion.String(),
				Name:       "notamachine1",
				Namespace:  "default",
			},
		},
	}

	resources := []runtime.Object{
		&host1, &host2, &host3, &host4, &host5,
		&machine1, &machine2,
		instance,
	}
	c := fakeclient.NewFakeClientWithScheme(scheme, resources...)
	reconciler := ReconcileMachineSet{
		Client: c,
		scheme: scheme,
	}

	// Run reconciliation
	_, err = reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: machinesetKey1})
	if err != nil {
		t.Errorf("%v", err)
	}

	// Check the MachineSet to see if it was scaled correctly
	ms := machinev1beta1.MachineSet{}
	err = c.Get(context.TODO(), machinesetKey1, &ms)
	switch {
	case err != nil:
		t.Errorf("%v", err)
	case ms.Spec.Replicas == nil:
		t.Logf("Replicas is nil")
		t.FailNow()
	case *ms.Spec.Replicas != 2:
		t.Logf("Replicas %d is not 2", *ms.Spec.Replicas)
		t.FailNow()
	}

	// Delete a host and expect the MachineSet to be scaled down
	err = c.Delete(context.TODO(), &host1)
	if err != nil {
		t.Errorf("%v", err)
	}

	_, err = reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: machinesetKey1})
	if err != nil {
		t.Errorf("%v", err)
	}

	ms = machinev1beta1.MachineSet{}
	err = c.Get(context.TODO(), machinesetKey1, &ms)
	if err != nil {
		t.Errorf("%v", err)
	}
	if ms.Spec.Replicas == nil || *ms.Spec.Replicas != 1 {
		t.Logf("Replicas is not 1")
		t.FailNow()
	}
}

// TestIgnore ensures that a MachineSet without the annotation gets ignored.
func TestIgnore(t *testing.T) {
	scheme := runtime.NewScheme()
	machinev1beta1.AddToScheme(scheme)
	bmoapis.AddToScheme(scheme)

	rawProviderSpec, err := json.Marshal(&bmv1alpha1.BareMetalMachineProviderSpec{})
	if err != nil {
		t.Errorf("%v", err)
	}

	// Since there are no BareMetalHosts, this would be scaled to 0 if it had
	// the annotation. Since it does not have the annotation, it should be
	// ignored.
	five := int32(5)
	instance := &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      machinesetKey2.Name,
			Namespace: machinesetKey2.Namespace,
		},
		Spec: machinev1beta1.MachineSetSpec{
			Replicas: &five,
			Template: machinev1beta1.MachineTemplateSpec{
				Spec: machinev1beta1.MachineSpec{
					ProviderSpec: machinev1beta1.ProviderSpec{
						Value: &runtime.RawExtension{Raw: rawProviderSpec},
					},
				},
			},
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{"machine.openshift.io/cluster-api-machineset": "cluster0-worker"},
			},
		},
	}

	c := fakeclient.NewFakeClientWithScheme(scheme, instance)
	reconciler := ReconcileMachineSet{
		Client: c,
		scheme: scheme,
	}

	// Run reconciliation
	_, err = reconciler.Reconcile(context.TODO(), reconcile.Request{NamespacedName: machinesetKey2})
	if err != nil {
		t.Errorf("%v", err)
	}

	// Verify that the MachineSet did not get scaled.
	ms := machinev1beta1.MachineSet{}
	err = c.Get(context.TODO(), machinesetKey2, &ms)
	if err != nil {
		t.Errorf("%v", err)
	}
	if *ms.Spec.Replicas != 5 {
		t.Logf("replicas is not 5; the MachineSet was not ignored as expected")
		t.FailNow()
	}
}
