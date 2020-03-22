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
	"fmt"
	bmh "github.com/metal3-io/baremetal-operator/pkg/apis/metal3/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"time"

	machinev1 "github.com/openshift/cluster-api/pkg/apis/machine/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	externalRemediationAnnotation = "host.metal3.io/external-remediation"
	bareMetalHostAnnotation       = "metal3.io/BareMetalHost"
	rebootAnnotation              = "reboot.metal3.io/machine-remediation"
	controllerName                = "machine-remediation-controller"
)

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Machine Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	r := newReconciler(mgr)
	return add(mgr, newReconciler(mgr), r.BareMetalHostToMachine)
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) *ReconcileMachineRemediation {
	return &ReconcileMachineRemediation{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler, baremetalhostToMachine handler.ToRequestsFunc) error {
	// Create a new controller
	c, err := controller.New(controllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Machine
	err = c.Watch(&source.Kind{Type: &machinev1.Machine{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return c.Watch(&source.Kind{Type: &bmh.BareMetalHost{}},
		&handler.EnqueueRequestsFromMapFunc{ToRequests: baremetalhostToMachine})

}

var _ reconcile.Reconciler = &ReconcileMachineRemediation{}
var mrLog = log.Log.WithName(controllerName)

// ReconcileMachineRemediation reconciles a Machine object
type ReconcileMachineRemediation struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Machine object and makes changes based on the state read
// and what is in the Machine.Spec
// +kubebuilder:rbac:groups=machine.openshift.io,resources=machines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=machine.openshift.io,resources=machines/status,verbs=get;update;patch
func (r *ReconcileMachineRemediation) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	machine := &machinev1.Machine{}

	err := r.Get(context.TODO(), request.NamespacedName, machine)
	if err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		mrLog.Error(err, "failed to retrieve Machine object")
		return reconcile.Result{}, err
	}

	host, err := r.getBareMetalHostByMachine(machine)

	if err != nil {
		mrLog.Error(err, "failed to get BareMetalHost from Machine")
		return reconcile.Result{}, err
	}

	needsRemediation := false
	if len(machine.Annotations) > 0 {
		_, needsRemediation = machine.Annotations[externalRemediationAnnotation]
	}

	if hasRebootAnnotation(host) && !host.Status.PoweredOn && !needsRemediation {
		mrLog.Info("Found powered off host, no remediation is required. Requesting power on",
			"Host name", host.Name, "Machine name", machine.Name)
		return r.requestPowerOn(host)
	}

	if !hasRebootAnnotation(host) && host.Status.PoweredOn && needsRemediation {
		mrLog.Info("Found powered on host, remediation needed. Requesting power off",
			"Host name", host.Name, "Machine name", machine.Name)
		return r.requestPowerOff(host)
	}

	node, err := r.getNodeByMachine(machine)

	if err != nil {
		if !errors.IsNotFound(err) {
			mrLog.Error(err, "failed to get Node from Machine", "Machine name", machine.Name)
			return reconcile.Result{}, err
		}
	}

	if hasRebootAnnotation(host) && !host.Status.PoweredOn && needsRemediation {
		if node != nil {
			mrLog.Info("Deleting node", "Node name", node.Name, "Machine name", machine.Name)
			return r.deleteNode(node)
		}

		mrLog.Info("Node is deleted. Remove remediation strategy annotation",
			"Machine name", machine.Name)
		return r.deleteExtRemediationAnnotation(machine)
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileMachineRemediation) BareMetalHostToMachine(obj handler.MapObject) []reconcile.Request {
	if host, ok := obj.Object.(*bmh.BareMetalHost); ok {
		if host.Spec.ConsumerRef != nil &&
			host.Spec.ConsumerRef.Kind == "Machine" &&
			host.Spec.ConsumerRef.APIVersion == machinev1.SchemeGroupVersion.String() {
			return []reconcile.Request{{NamespacedName: types.NamespacedName{
				Name:      host.Spec.ConsumerRef.Name,
				Namespace: host.Spec.ConsumerRef.Namespace,
			}}}
		}
	}
	return []reconcile.Request{}
}

//deleteExtRemediationAnnotation deletes remediation strategy annotations
func (r *ReconcileMachineRemediation) deleteExtRemediationAnnotation(machine *machinev1.Machine) (reconcile.Result, error) {
	if len(machine.Annotations) == 0 {
		return reconcile.Result{}, nil
	}

	delete(machine.Annotations, externalRemediationAnnotation)

	if err := r.Update(context.TODO(), machine); err != nil {
		mrLog.Error(err, "failed to delete remediation annotation", "Machine name", machine.Name)
		return reconcile.Result{}, err
	}

	return reconcile.Result{Requeue: true}, nil
}

//hasRebootAnnotation checks if the reboot annotation exist on the baremetalhost
func hasRebootAnnotation(baremetalhost *bmh.BareMetalHost) bool {
	if len(baremetalhost.Annotations) == 0 {
		return false
	}
	_, exists := baremetalhost.Annotations[rebootAnnotation]

	return exists
}

//requestPowerOn removes reboot annotation on baremetalhost which signal BMO to power on the machine
func (r *ReconcileMachineRemediation) requestPowerOff(baremetalhost *bmh.BareMetalHost) (reconcile.Result, error) {
	if baremetalhost.Annotations == nil {
		baremetalhost.Annotations = make(map[string]string)
	}

	baremetalhost.Annotations[rebootAnnotation] = ""

	if err := r.Update(context.TODO(), baremetalhost); err != nil {
		mrLog.Error(err, "failed to add reboot annotation", "Baremetalhost name", baremetalhost.Name)
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
}

//requestPowerOn adds reboot annotation on baremetalhost which signal BMO to power off the machine
func (r *ReconcileMachineRemediation) requestPowerOn(baremetalhost *bmh.BareMetalHost) (reconcile.Result, error) {
	if baremetalhost.Annotations == nil {
		return reconcile.Result{Requeue: true}, nil
	}

	if _, rebootPending := baremetalhost.Annotations[rebootAnnotation]; !rebootPending {
		return reconcile.Result{Requeue: true}, nil
	}

	delete(baremetalhost.Annotations, rebootAnnotation)

	if err := r.Client.Update(context.TODO(), baremetalhost); err != nil {
		mrLog.Error(err, "failed to remove reboot annotation")
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: 10 * time.Second}, nil
}

// deleteMachineNode deletes the node that mapped to specified machine
func (r *ReconcileMachineRemediation) deleteNode(node *corev1.Node) (reconcile.Result, error) {
	if err := r.Delete(context.TODO(), node); err != nil {
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: true}, nil
		}
		mrLog.Error(err, "failed to delete Node", "Node name", node.Name)
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

// getNodeByMachine returns the node object referenced by machine
func (r *ReconcileMachineRemediation) getNodeByMachine(machine *machinev1.Machine) (*corev1.Node, error) {
	if machine.Status.NodeRef == nil {
		return nil, errors.NewNotFound(corev1.Resource("ObjectReference"), machine.Name)
	}

	node := &corev1.Node{}
	key := client.ObjectKey{
		Name:      machine.Status.NodeRef.Name,
		Namespace: machine.Status.NodeRef.Namespace,
	}

	if err := r.Client.Get(context.TODO(), key, node); err != nil {
		return nil, err
	}
	return node, nil
}

// getBareMetalHostByMachine returns the bare metal host that linked to the machine
func (r *ReconcileMachineRemediation) getBareMetalHostByMachine(machine *machinev1.Machine) (*bmh.BareMetalHost, error) {
	bmhKey, ok := machine.Annotations[bareMetalHostAnnotation]
	if !ok {
		return nil, fmt.Errorf("machine does not have bare metal host annotation")
	}

	bmhNamespace, bmhName, err := cache.SplitMetaNamespaceKey(bmhKey)
	baremetalhost := &bmh.BareMetalHost{}
	key := client.ObjectKey{
		Name:      bmhName,
		Namespace: bmhNamespace,
	}

	err = r.Client.Get(context.TODO(), key, baremetalhost)
	if err != nil {
		return nil, err
	}
	return baremetalhost, nil
}
