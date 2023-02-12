/*
Copyright 2020 The Kubernetes Authors.

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

package baremetal

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	bmov1alpha1 "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
)

const (
	powerOffAnnotation              = "reboot.metal3.io/metal3-remediation-%s"
	nodeAnnotationsBackupAnnotation = "remediation.metal3.io/node-annotations-backup"
	nodeLabelsBackupAnnotation      = "remediation.metal3.io/node-labels-backup"
	// HostAnnotation is the key for an annotation that should go on a Metal3Machine to
	// reference what BareMetalHost it corresponds to.
	HostAnnotation    = "metal3.io/BareMetalHost"
	machineRoleLabel  = "machine.openshift.io/cluster-api-machine-role"
	machineRoleMaster = "master"
)

// RemediationManagerInterface is an interface for a RemediationManager.
type RemediationManagerInterface interface {
	SetFinalizer()
	UnsetFinalizer()
	HasFinalizer() bool
	TimeToRemediate(timeout time.Duration) (bool, time.Duration)
	SetPowerOffAnnotation(ctx context.Context) error
	RemovePowerOffAnnotation(ctx context.Context) error
	IsPowerOffRequested(ctx context.Context) (bool, error)
	IsPoweredOn(ctx context.Context) (bool, error)
	SetUnhealthyAnnotation(ctx context.Context) error
	GetUnhealthyHost(ctx context.Context) (*bmov1alpha1.BareMetalHost, *patch.Helper, error)
	OnlineStatus(host *bmov1alpha1.BareMetalHost) bool
	GetRemediationType() infrav1.RemediationType
	RetryLimitIsSet() bool
	HasReachRetryLimit() bool
	SetRemediationPhase(phase string)
	GetRemediationPhase() string
	GetLastRemediatedTime() *metav1.Time
	SetLastRemediationTime(remediationTime *metav1.Time)
	GetTimeout() *metav1.Duration
	IncreaseRetryCount()
	GetNode(ctx context.Context) (*corev1.Node, error)
	UpdateNode(ctx context.Context, node *corev1.Node) error
	DeleteNode(ctx context.Context, node *corev1.Node) error
	SetNodeBackupAnnotations(annotations string, labels string) bool
	GetNodeBackupAnnotations() (annotations, labels string)
	RemoveNodeBackupAnnotations()

	// OCP specific methods, differs from upstream metal3

	CanReprovision(context.Context) (bool, error)
	DeleteMachine(ctx context.Context) error
}

// RemediationManager is responsible for performing remediation reconciliation.
type RemediationManager struct {
	Client            client.Client
	Metal3Remediation *infrav1.Metal3Remediation
	OCPMachine        *machinev1beta1.Machine
	Log               logr.Logger
}

// enforce implementation of interface.
var _ RemediationManagerInterface = &RemediationManager{}

// NewRemediationManager returns a new helper for managing a Metal3Remediation object.
func NewRemediationManager(client client.Client,
	metal3remediation *infrav1.Metal3Remediation, ocpMachine *machinev1beta1.Machine,
	remediationLog logr.Logger) (*RemediationManager, error) {
	return &RemediationManager{
		Client:            client,
		Metal3Remediation: metal3remediation,
		OCPMachine:        ocpMachine,
		Log:               remediationLog,
	}, nil
}

// SetFinalizer sets finalizer. Return if it was set.
func (r *RemediationManager) SetFinalizer() {
	controllerutil.AddFinalizer(r.Metal3Remediation, infrav1.RemediationFinalizer)
}

// UnsetFinalizer unsets finalizer.
func (r *RemediationManager) UnsetFinalizer() {
	controllerutil.RemoveFinalizer(r.Metal3Remediation, infrav1.RemediationFinalizer)
}

// HasFinalizer returns if finalizer is set.
func (r *RemediationManager) HasFinalizer() bool {
	return controllerutil.ContainsFinalizer(r.Metal3Remediation, infrav1.RemediationFinalizer)
}

// TimeToRemediate checks if it is time to execute a next remediation step
// and returns seconds to next remediation time.
func (r *RemediationManager) TimeToRemediate(timeout time.Duration) (bool, time.Duration) {
	now := time.Now()

	// status is not updated yet
	if r.Metal3Remediation.Status.LastRemediated == nil {
		return false, timeout
	}

	if r.Metal3Remediation.Status.LastRemediated.Add(timeout).Before(now) {
		return true, time.Duration(0)
	}

	lastRemediated := now.Sub(r.Metal3Remediation.Status.LastRemediated.Time)
	nextRemediation := timeout - lastRemediated + time.Second
	return false, nextRemediation
}

// SetPowerOffAnnotation sets poweroff annotation on unhealthy host.
func (r *RemediationManager) SetPowerOffAnnotation(ctx context.Context) error {
	host, helper, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		return err
	}
	if host == nil {
		return errors.New("Unable to set a PowerOff Annotation, Host not found")
	}

	r.Log.Info("Adding PowerOff annotation to host", "host", host.Name)
	rebootMode := bmov1alpha1.RebootAnnotationArguments{}
	rebootMode.Mode = bmov1alpha1.RebootModeHard
	marshalledMode, err := json.Marshal(rebootMode)

	if err != nil {
		return err
	}

	if host.Annotations == nil {
		host.Annotations = make(map[string]string)
	}
	host.Annotations[r.getPowerOffAnnotationKey()] = string(marshalledMode)
	return helper.Patch(ctx, host)
}

// RemovePowerOffAnnotation removes poweroff annotation from unhealthy host.
func (r *RemediationManager) RemovePowerOffAnnotation(ctx context.Context) error {
	host, helper, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		return err
	}
	if host == nil {
		return errors.New("Unable to remove PowerOff Annotation, Host not found")
	}

	r.Log.Info("Removing PowerOff annotation from host", "host name", host.Name)
	delete(host.Annotations, r.getPowerOffAnnotationKey())
	return helper.Patch(ctx, host)
}

// IsPowerOffRequested returns true if poweroff annotation is set.
func (r *RemediationManager) IsPowerOffRequested(ctx context.Context) (bool, error) {
	host, _, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		return false, err
	}
	if host == nil {
		return false, errors.New("Unable to check PowerOff Annotation, Host not found")
	}

	if _, ok := host.Annotations[r.getPowerOffAnnotationKey()]; ok {
		return true, nil
	}
	return false, nil
}

// IsPoweredOn returns true if the host is powered on.
func (r *RemediationManager) IsPoweredOn(ctx context.Context) (bool, error) {
	host, _, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		return false, err
	}
	if host == nil {
		return false, errors.New("Unable to check power status, Host not found")
	}

	return host.Status.PoweredOn, nil
}

// SetUnhealthyAnnotation sets capm3.UnhealthyAnnotation on unhealthy host.
func (r *RemediationManager) SetUnhealthyAnnotation(ctx context.Context) error {
	host, helper, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		return err
	}
	if host == nil {
		return errors.New("Unable to set an Unhealthy Annotation, Host not found")
	}

	r.Log.Info("Adding Unhealthy annotation to host", "host", host.Name)
	if host.Annotations == nil {
		host.Annotations = make(map[string]string, 1)
	}
	host.Annotations[infrav1.UnhealthyAnnotation] = "capm3/UnhealthyNode"
	return helper.Patch(ctx, host)
}

// GetUnhealthyHost gets the associated host for unhealthy machine. Returns nil if not found. Assumes the
// host is in the same namespace as the unhealthy machine.
func (r *RemediationManager) GetUnhealthyHost(ctx context.Context) (*bmov1alpha1.BareMetalHost, *patch.Helper, error) {
	host, err := getUnhealthyHost(ctx, r.OCPMachine, r.Client, r.Log)
	if err != nil || host == nil {
		return host, nil, err
	}
	helper, err := patch.NewHelper(host, r.Client)
	return host, helper, err
}

func getUnhealthyHost(ctx context.Context, ocpMachine *machinev1beta1.Machine, cl client.Client,
	rLog logr.Logger,
) (*bmov1alpha1.BareMetalHost, error) {
	annotations := ocpMachine.ObjectMeta.GetAnnotations()
	if annotations == nil {
		err := fmt.Errorf("unable to get %s annotations", ocpMachine.Name)
		return nil, err
	}
	hostKey, ok := annotations[HostAnnotation]
	if !ok {
		err := fmt.Errorf("unable to get %s HostAnnotation", ocpMachine.Name)
		return nil, err
	}
	hostNamespace, hostName, err := cache.SplitMetaNamespaceKey(hostKey)
	if err != nil {
		rLog.Error(err, "Error parsing annotation value", "annotation key", hostKey)
		return nil, err
	}

	host := bmov1alpha1.BareMetalHost{}
	key := client.ObjectKey{
		Name:      hostName,
		Namespace: hostNamespace,
	}
	err = cl.Get(ctx, key, &host)
	if apierrors.IsNotFound(err) {
		rLog.Info("Annotated host not found", "host", hostKey)
		return nil, err
	} else if err != nil {
		return nil, err
	}
	return &host, nil
}

// OnlineStatus returns hosts Online field value.
func (r *RemediationManager) OnlineStatus(host *bmov1alpha1.BareMetalHost) bool {
	return host.Spec.Online
}

// GetRemediationType return type of remediation strategy.
func (r *RemediationManager) GetRemediationType() infrav1.RemediationType {
	if r.Metal3Remediation.Spec.Strategy == nil {
		return ""
	}
	return r.Metal3Remediation.Spec.Strategy.Type
}

// RetryLimitIsSet returns true if retryLimit is set, false if not.
func (r *RemediationManager) RetryLimitIsSet() bool {
	if r.Metal3Remediation.Spec.Strategy == nil {
		return false
	}
	return r.Metal3Remediation.Spec.Strategy.RetryLimit > 0
}

// HasReachRetryLimit returns true if retryLimit is reached.
func (r *RemediationManager) HasReachRetryLimit() bool {
	if r.Metal3Remediation.Spec.Strategy == nil {
		return false
	}
	return r.Metal3Remediation.Spec.Strategy.RetryLimit == r.Metal3Remediation.Status.RetryCount
}

// SetRemediationPhase setting the state of the remediation.
func (r *RemediationManager) SetRemediationPhase(phase string) {
	r.Log.Info("Switching remediation phase", "remediationPhase", phase)
	r.Metal3Remediation.Status.Phase = phase
}

// GetRemediationPhase returns current status of the remediation.
func (r *RemediationManager) GetRemediationPhase() string {
	return r.Metal3Remediation.Status.Phase
}

// GetLastRemediatedTime returns last remediation time.
func (r *RemediationManager) GetLastRemediatedTime() *metav1.Time {
	return r.Metal3Remediation.Status.LastRemediated
}

// SetLastRemediationTime setting last remediation timestamp on Status.
func (r *RemediationManager) SetLastRemediationTime(remediationTime *metav1.Time) {
	r.Log.Info("Last remediation time", "remediationTime", remediationTime)
	r.Metal3Remediation.Status.LastRemediated = remediationTime
}

// GetTimeout returns timeout duration from remediation request Spec.
func (r *RemediationManager) GetTimeout() *metav1.Duration {
	return r.Metal3Remediation.Spec.Strategy.Timeout
}

// IncreaseRetryCount increases the retry count on Status.
func (r *RemediationManager) IncreaseRetryCount() {
	r.Metal3Remediation.Status.RetryCount++
}

// GetNode returns the Node associated with the machine in the current context.
func (r *RemediationManager) GetNode(ctx context.Context) (*corev1.Node, error) {

	if r.OCPMachine.Status.NodeRef == nil {
		r.Log.Error(nil, "metal3Remediation's node could not be retrieved, machine's nodeRef is nil")
		return nil, errors.Errorf("metal3Remediation's node could not be retrieved, machine's nodeRef is nil")
	}

	node := &corev1.Node{}
	key := client.ObjectKey{Name: r.OCPMachine.Status.NodeRef.Name}
	err := r.Client.Get(ctx, key, node)
	if apierrors.IsNotFound(err) {
		return nil, nil
	} else if err != nil {
		r.Log.Error(err, "Could not get cluster node")
		return nil, errors.Wrapf(err, "Could not get cluster node")
	}
	return node, nil
}

// UpdateNode updates the given node.
func (r *RemediationManager) UpdateNode(ctx context.Context, node *corev1.Node) error {
	err := r.Client.Update(ctx, node)
	if err != nil {
		r.Log.Error(err, "Could not update cluster node")
		return errors.Wrapf(err, "Could not update cluster node")
	}
	return nil
}

// DeleteNode deletes the given node.
func (r *RemediationManager) DeleteNode(ctx context.Context, node *corev1.Node) error {
	if !node.DeletionTimestamp.IsZero() {
		return nil
	}

	err := r.Client.Delete(ctx, node)
	if err != nil {
		r.Log.Error(err, "Could not delete cluster node")
		return errors.Wrapf(err, "Could not delete cluster node")
	}
	return nil
}

// SetNodeBackupAnnotations sets the given node annotations and labels as remediation annotations.
// Returns whether annotations were set or modified, or not.
func (r *RemediationManager) SetNodeBackupAnnotations(annotations string, labels string) bool {
	rem := r.Metal3Remediation
	if rem.Annotations == nil {
		rem.Annotations = make(map[string]string)
	}
	if rem.Annotations[nodeAnnotationsBackupAnnotation] != annotations ||
		rem.Annotations[nodeLabelsBackupAnnotation] != labels {
		rem.Annotations[nodeAnnotationsBackupAnnotation] = annotations
		rem.Annotations[nodeLabelsBackupAnnotation] = labels
		return true
	}
	return false
}

// GetNodeBackupAnnotations gets the stringified annotations and labels from the remediation annotations.
func (r *RemediationManager) GetNodeBackupAnnotations() (annotations, labels string) {
	rem := r.Metal3Remediation
	if rem.Annotations == nil {
		return "", ""
	}
	annotations = rem.Annotations[nodeAnnotationsBackupAnnotation]
	labels = rem.Annotations[nodeLabelsBackupAnnotation]
	return
}

// RemoveNodeBackupAnnotations removes the node backup annotation from the remediation resource.
func (r *RemediationManager) RemoveNodeBackupAnnotations() {
	rem := r.Metal3Remediation
	if rem.Annotations == nil {
		return
	}
	delete(rem.Annotations, nodeAnnotationsBackupAnnotation)
	delete(rem.Annotations, nodeLabelsBackupAnnotation)
}

// getPowerOffAnnotationKey returns the key of the power off annotation.
func (r *RemediationManager) getPowerOffAnnotationKey() string {
	return fmt.Sprintf(powerOffAnnotation, r.Metal3Remediation.UID)
}

func (r *RemediationManager) CanReprovision(ctx context.Context) (bool, error) {
	baremetalhost, _, err := r.GetUnhealthyHost(ctx)
	if err != nil {
		r.Log.Error(err, "Failed to get BMH for machine", "machine", r.OCPMachine.Name)
		return false, err
	}
	if baremetalhost.Spec.ExternallyProvisioned {
		r.Log.Info("Reprovisioning of machine not allowed: BMH is externally provisioned", "machine", r.OCPMachine.Name, "bmh", baremetalhost.Name)
		return false, nil
	}
	if metav1.GetControllerOf(r.OCPMachine) == nil {
		r.Log.Info("Reprovisioning of machine not allowed: no owning controller", "machine", r.OCPMachine.Name)
		return false, nil
	}
	if r.OCPMachine.Labels[machineRoleLabel] == machineRoleMaster {
		r.Log.Info("Reprovisioning of machine not allowed: has master role", "machine", r.OCPMachine.Name)
		return false, nil
	}
	return true, nil
}

func (r *RemediationManager) DeleteMachine(ctx context.Context) error {
	err := r.Client.Delete(ctx, r.OCPMachine)
	if err != nil {
		r.Log.Error(err, "Failed to delete machine", "machine", r.OCPMachine.Name)
	}
	return err
}
