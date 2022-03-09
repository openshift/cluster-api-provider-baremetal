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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMapper(t *testing.T) {
	var mo client.Object
	scheme := runtime.NewScheme()
	machinev1beta1.AddToScheme(scheme)
	bmoapis.AddToScheme(scheme)

	testCases := []struct {
		Host          *bmh.BareMetalHost
		Annotations   map[string]string
		ExpectRequest bool
		FailMessage   string
	}{
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
			},
			Annotations:   map[string]string{AutoScaleAnnotation: "yesplease"},
			ExpectRequest: true,
			FailMessage:   "host with annotation and matching label should generate a request",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "medium"},
				},
			},
			Annotations:   map[string]string{AutoScaleAnnotation: "yesplease"},
			ExpectRequest: false,
			FailMessage:   "host with annotation and non-matching label should not generate a request",
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "default",
					Labels:    map[string]string{"size": "large"},
				},
			},
			Annotations:   map[string]string{},
			ExpectRequest: false,
			FailMessage:   "host without annotation should not generate a request",
		},
	}

	for _, tc := range testCases {
		ms, err := newMachineSet(tc.Annotations)
		if err != nil {
			t.Errorf("%v", err)
		}
		c := fakeclient.NewFakeClientWithScheme(scheme, ms, tc.Host)
		mapper := msmapper{client: c}

		mo = tc.Host
		requests := mapper.Map(mo)

		if tc.ExpectRequest {
			if len(requests) != 1 {
				t.Logf("expected 1 request, got %d", len(requests))
				t.Logf(tc.FailMessage)
				t.FailNow()
			}
			req := requests[0]
			if req.NamespacedName.Name != ms.Name {
				t.Logf("expected Name %s, got %s", req.NamespacedName.Name, ms.Name)
				t.Logf(tc.FailMessage)
				t.FailNow()
			}
			if req.NamespacedName.Namespace != ms.Namespace {
				t.Logf("expected NameSpace %s, got %s", req.NamespacedName.Namespace, ms.Namespace)
				t.Logf(tc.FailMessage)
				t.FailNow()
			}
		} else if len(requests) != 0 {
			t.Logf("expected 0 requests, got %d", len(requests))
			t.Logf(tc.FailMessage)
			t.FailNow()
		}
	}
}

func newMachineSet(annotations map[string]string) (*machinev1beta1.MachineSet, error) {
	rawProviderSpec, err := json.Marshal(&bmv1alpha1.BareMetalMachineProviderSpec{
		HostSelector: bmv1alpha1.HostSelector{
			MatchLabels: map[string]string{"size": "large"},
		},
	})
	if err != nil {
		return nil, err
	}
	two := int32(2)
	return &machinev1beta1.MachineSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        machinesetKey1.Name,
			Namespace:   machinesetKey1.Namespace,
			Annotations: annotations,
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
			Replicas: &two,
		},
	}, nil
}
