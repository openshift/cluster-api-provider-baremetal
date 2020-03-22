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

package machine_remediation

import (
	"context"
	bmh "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	machinev1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"testing"
)

const (
	defaultNamespace = "default"
)
var c = fakeclient.NewFakeClientWithScheme(scheme.Scheme)

func newTestReconciler() *ReconcileMachineRemediation {
	return &ReconcileMachineRemediation{
		Client: c,
		scheme: scheme.Scheme,
	}
}

func newRequest(machine *machinev1.Machine) reconcile.Request {
	namespacedName := types.NamespacedName{
		Namespace: machine.ObjectMeta.Namespace,
		Name:      machine.ObjectMeta.Name,
	}
	return reconcile.Request{NamespacedName: namespacedName}
}

var MachineSchemeGroupVersion = machinev1.SchemeGroupVersion

var (
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

// Adds the list of known types to Scheme.
func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(MachineSchemeGroupVersion, &machinev1.Machine{})
	metav1.AddToGroupVersion(scheme, MachineSchemeGroupVersion)
	scheme.AddKnownTypes(bmh.SchemeGroupVersion, &bmh.BareMetalHost{})
	metav1.AddToGroupVersion(scheme, bmh.SchemeGroupVersion)
	return nil
}

func tryReconcile(machine *machinev1.Machine, t *testing.T) {

	r := newTestReconciler()

	_, err := r.Reconcile(newRequest(machine))
	if err != nil {
		t.Log("Error from reconcile: ", err)
		t.Fail()
	}

}

func TestFullFlow(t *testing.T) {
	AddToScheme(scheme.Scheme)

	machine, machineNamespacedName := getMachine()
	host, hostNamespacedName := getBareMetalHost()
	node, nodeNamespacedName := geteNode()
	linkMachineAndNode(machine, node)
	host.Status.PoweredOn = true

	//starting test with machine that needs remediation
	machine.Annotations = make(map[string]string)
	machine.Annotations[bareMetalHostAnnotation] = host.Namespace + "/" + host.Name
	machine.Annotations[externalRemediationAnnotation] = ""

	c.Create(context.TODO(), machine)
	c.Create(context.TODO(), host)
	c.Create(context.TODO(), node)

	tryReconcile(machine, t)

	host = &bmh.BareMetalHost{}
	c.Get(context.TODO(), hostNamespacedName, host)
	if !hasRebootAnnotation(host) {
		t.Log("Expected reboot annotation on the host but none found")
		t.Fail()
	}

	host.Status.PoweredOn = false
	c.Update(context.TODO(), host)

	tryReconcile(machine, t)

	machine = &machinev1.Machine{}
	c.Get(context.TODO(), machineNamespacedName, machine)

	//tryReconcile(machine, t)

	node = &corev1.Node{}
	err := c.Get(context.TODO(), nodeNamespacedName, node)
	if !errors.IsNotFound(err) {
		t.Log("Expected node to be deleted")
		t.Fail()
	}

	tryReconcile(machine, t)

	machine = &machinev1.Machine{}
	c.Get(context.TODO(), machineNamespacedName, machine)

	if len(machine.Annotations) > 0 {
		if _, exists := machine.Annotations[externalRemediationAnnotation]; exists {
			t.Log("Expected external remediation annotation to be removed but it's still there")
			t.Fail()
		}
	}

	tryReconcile(machine, t)

	host = &bmh.BareMetalHost{}
	c.Get(context.TODO(), hostNamespacedName, host)

	if hasRebootAnnotation(host) {
		t.Log("Expected reboot annotation to be removed but it's still there")
		t.Fail()
	}

	host.Status.PoweredOn = true
	c.Update(context.TODO(), host)

	tryReconcile(machine, t)

	machine = &machinev1.Machine{}
	c.Get(context.TODO(), machineNamespacedName, machine)

	node, _ = geteNode()
	linkMachineAndNode(machine,node)
	c.Create(context.TODO(), node)
	c.Update(context.TODO(), machine)

	//make sure nothing happens after remediation
	tryReconcile(machine, t)
	tryReconcile(machine, t)

	host = &bmh.BareMetalHost{}
	c.Get(context.TODO(), hostNamespacedName, host)

	if hasRebootAnnotation(host) {
		t.Log("Expected reboot annotation to be removed but it's still there, maybe reboot loops?")
		t.Fail()
	}

}

func geteNode() (*corev1.Node, types.NamespacedName) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{
		Name:      "node1",
		Namespace: defaultNamespace,
	},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: corev1.SchemeGroupVersion.String(),
		}}

	nodeNamespacedName := types.NamespacedName{
		Namespace: node.ObjectMeta.Namespace,
		Name:      node.ObjectMeta.Name,
	}

	return node, nodeNamespacedName
}

func getBareMetalHost() (*bmh.BareMetalHost, types.NamespacedName) {
	host := &bmh.BareMetalHost{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "host1",
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "BareMetalHost",
			APIVersion: bmh.SchemeGroupVersion.String(),
		},
		Spec:   bmh.BareMetalHostSpec{},
		Status: bmh.BareMetalHostStatus{},
	}

	hostNamespacedName := types.NamespacedName{
		Namespace: host.ObjectMeta.Namespace,
		Name:      host.ObjectMeta.Name,
	}

	return host, hostNamespacedName
}

func getMachine() (*machinev1.Machine, types.NamespacedName) {
	machine := &machinev1.Machine{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "machine1",
			Namespace: defaultNamespace,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Machine",
			APIVersion: machinev1.SchemeGroupVersion.String(),
		},
		Status: machinev1.MachineStatus{},
	}

	machineNamespacedName := types.NamespacedName{
		Namespace: machine.ObjectMeta.Namespace,
		Name:      machine.ObjectMeta.Name,
	}

	return machine , machineNamespacedName
}

func linkMachineAndNode(machine *machinev1.Machine , node *corev1.Node) {
	machine.Status.NodeRef = &corev1.ObjectReference{
		Kind:      "Node",
		Namespace: defaultNamespace,
		Name:      node.Name,
	}
}