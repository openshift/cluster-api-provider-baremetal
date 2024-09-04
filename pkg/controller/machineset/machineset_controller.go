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
	"context"
	"fmt"
	"reflect"

	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	actuator "github.com/openshift/cluster-api-provider-baremetal/pkg/cloud/baremetal/actuators/machine"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("machineset-controller")

// errConsumerNotFound indicates that the Machine referenced in a BareMetalHost's
// Spec.ConsumerRef cannot be found. That's an unexpected state, so we don't
// want to do any scaling until it gets resolved.
var errConsumerNotFound = fmt.Errorf("consuming Machine not found")

// Add creates a new MachineSet Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileMachineSet{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("machineset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to MachineSet
	err = c.Watch(source.Kind(mgr.GetCache(), &machinev1beta1.MachineSet{},
		&handler.TypedEnqueueRequestForObject[*machinev1beta1.MachineSet]{}, TypedResourceVersionChangedPredicate[*machinev1beta1.MachineSet]{}))
	if err != nil {
		return err
	}
	mapper := msmapper{client: mgr.GetClient()}
	err = c.Watch(source.Kind(mgr.GetCache(), &bmh.BareMetalHost{},
		handler.TypedEnqueueRequestsFromMapFunc(mapper.Map), TypedResourceVersionChangedPredicate[*bmh.BareMetalHost]{}))
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileMachineSet{}

// ReconcileMachineSet reconciles a MachineSet object
type ReconcileMachineSet struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a MachineSet object and makes changes based on the state read
// and what is in the MachineSet.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=cluster.k8s.io,resources=machinesets,verbs=get;list;watch;update;patch
func (r *ReconcileMachineSet) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log := log.WithValues("MachineSet", request.NamespacedName.String())
	// Fetch the MachineSet instance
	instance := &machinev1beta1.MachineSet{}
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Don't take action if the annotation is not present
	annotations := instance.ObjectMeta.GetAnnotations()
	if annotations == nil {
		return reconcile.Result{}, nil
	}
	_, present := annotations[AutoScaleAnnotation]
	if !present {
		return reconcile.Result{}, nil
	}

	// Make sure the MachineSet has a non-empty selector.
	msselector, err := metav1.LabelSelectorAsSelector(&instance.Spec.Selector)
	if err != nil {
		return reconcile.Result{}, err
	}
	if msselector.Empty() {
		// The cluster-api machinesetcontroller expects every MachineSet to have
		// its Selector set.
		log.Info("MachineSet has empty selector, which is unexpected. Will not attempt scaling.")
		return reconcile.Result{}, nil
	}

	hostselector, err := actuator.SelectorFromProviderSpec(&instance.Spec.Template.Spec.ProviderSpec)
	if err != nil {
		return reconcile.Result{}, err
	}

	hosts := &bmh.BareMetalHostList{}
	opts := &client.ListOptions{
		Namespace: instance.Namespace,
	}

	err = r.List(ctx, hosts, opts)
	if err != nil {
		return reconcile.Result{}, err
	}

	var count int32
	for i := range hosts.Items {
		matches, err := r.hostMatches(ctx, hostselector, msselector, &hosts.Items[i])
		switch {
		case err == errConsumerNotFound:
			log.Info("Will not scale while BareMetalHost's consuming Machine is not found", "BareMetalHost.Name", &hosts.Items[i].Name)
			return reconcile.Result{}, nil
		case err != nil:
			return reconcile.Result{}, err
		case matches == true:
			count++
		}
	}

	if instance.Spec.Replicas == nil || count != *instance.Spec.Replicas {
		log.Info("Scaling MachineSet", "new_replicas", count, "old_replicas", instance.Spec.Replicas)
		new := instance.DeepCopy()
		new.Spec.Replicas = &count
		err = r.Update(ctx, new)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

// hostMatches returns true if the BareMetalHost matches the MachineSet.
func (r *ReconcileMachineSet) hostMatches(ctx context.Context, hostselector labels.Selector,
	msselector labels.Selector, host *bmh.BareMetalHost) (bool, error) {
	consumer := host.Spec.ConsumerRef

	if consumer == nil {
		// BMH is not consumed, so just see if it matches the host selector
		return hostselector.Matches(labels.Set(host.ObjectMeta.Labels)), nil
	}

	// We will only count this host if it is consumed by a Machine that
	// is part of the current MachineSet.
	machine := &machinev1beta1.Machine{}
	if consumer.Kind != "Machine" || consumer.APIVersion != machinev1beta1.SchemeGroupVersion.String() {
		// this host is being consumed by something else; don't count it
		return false, nil
	}

	// host is being consumed. Let's get the Machine and see if it
	// matches the current MachineSet.
	nn := types.NamespacedName{
		Name:      consumer.Name,
		Namespace: consumer.Namespace,
	}
	err := r.Get(ctx, nn, machine)
	if err != nil {
		if errors.IsNotFound(err) {
			return false, errConsumerNotFound
		}
		return false, err
	}
	return msselector.Matches(labels.Set(machine.ObjectMeta.Labels)), nil
}

// Bring in the TypedResourceVersionChangedPredicate from controller-runtime 0.19.0
// This  avoids bumping MAO and all of it's operands to the same version.

// ResourceVersionChangedPredicate implements a default update predicate function on resource version change.
type ResourceVersionChangedPredicate = TypedResourceVersionChangedPredicate[client.Object]

// TypedResourceVersionChangedPredicate implements a default update predicate function on resource version change.
type TypedResourceVersionChangedPredicate[T metav1.Object] struct {
	predicate.TypedFuncs[T]
}

// Update implements default UpdateEvent filter for validating resource version change.
func (TypedResourceVersionChangedPredicate[T]) Update(e event.TypedUpdateEvent[T]) bool {
	if isNil(e.ObjectOld) {
		log.Error(nil, "Update event has no old object to update", "event", e)
		return false
	}
	if isNil(e.ObjectNew) {
		log.Error(nil, "Update event has no new object to update", "event", e)
		return false
	}

	return e.ObjectNew.GetResourceVersion() != e.ObjectOld.GetResourceVersion()
}

func isNil(arg any) bool {
	if v := reflect.ValueOf(arg); !v.IsValid() || ((v.Kind() == reflect.Ptr ||
		v.Kind() == reflect.Interface ||
		v.Kind() == reflect.Slice ||
		v.Kind() == reflect.Map ||
		v.Kind() == reflect.Chan ||
		v.Kind() == reflect.Func) && v.IsNil()) {
		return true
	}
	return false
}
