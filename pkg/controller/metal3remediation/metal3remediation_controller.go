/*
Copyright The Kubernetes Authors.

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

package metal3remediation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"

	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"github.com/openshift/cluster-api-provider-baremetal/pkg/baremetal"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Metal3RemediationReconciler reconciles a Metal3Remediation object.
type Metal3RemediationReconciler struct {
	client.Client
	ManagerFactory baremetal.ManagerFactoryInterface
	Log            logr.Logger
}

// Generate RBAC
//go:generate go run ../../../vendor/sigs.k8s.io/controller-tools/cmd/controller-gen paths=./... rbac:roleName=machine-api-controllers-baremetal crd output:dir:=./../../../config/rbac

// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3remediations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=infrastructure.cluster.x-k8s.io,resources=metal3remediations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=machine.openshift.io,resources=machines;machines/status,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch;delete

// +kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts/status,verbs=get;update;patch

// Reconcile handles Metal3Remediation events.
func (r *Metal3RemediationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, rerr error) {
	remediationLog := r.Log.WithValues("metal3remediation", req.NamespacedName)

	// Fetch the Metal3Remediation instance.
	metal3Remediation := &infrav1.Metal3Remediation{}
	if err := r.Client.Get(ctx, req.NamespacedName, metal3Remediation); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		remediationLog.Error(err, "unable to get metal3Remediation")
		return ctrl.Result{}, err
	}

	helper, err := patch.NewHelper(metal3Remediation, r.Client)
	if err != nil {
		remediationLog.Error(err, "failed to init patch helper")
		return ctrl.Result{}, err
	}

	defer func() {
		// Always attempt to Patch the Remediation object and status after each reconciliation.
		// Patch ObservedGeneration only if the reconciliation completed successfully
		patchOpts := []patch.Option{}
		patchOpts = append(patchOpts, patch.WithStatusObservedGeneration{})

		patchErr := helper.Patch(ctx, metal3Remediation, patchOpts...)
		if patchErr != nil {
			remediationLog.Error(patchErr, "failed to Patch metal3Remediation")
			// trigger requeue!
			rerr = patchErr
		}
	}()

	ocpMachine, err := r.getMachine(remediationLog, metal3Remediation)
	if err != nil {
		return ctrl.Result{}, err
	}

	remediationLog = remediationLog.WithValues("ocpmachine", ocpMachine.Name)

	// Create a helper for managing the remediation object.
	remediationMgr, err := r.ManagerFactory.NewRemediationManager(metal3Remediation, ocpMachine, remediationLog)
	if err != nil {
		remediationLog.Error(err, "failed to create helper for managing the metal3remediation")
		return ctrl.Result{}, errors.Wrapf(err, "failed to create helper for managing the metal3remediation")
	}

	// Handle both deleted and non-deleted remediations
	return r.reconcileNormal(ctx, remediationMgr)
}

func (r *Metal3RemediationReconciler) reconcileNormal(ctx context.Context,
	remediationMgr baremetal.RemediationManagerInterface,
) (ctrl.Result, error) {
	// If host is gone, exit early
	host, _, err := remediationMgr.GetUnhealthyHost(ctx)
	if err != nil {
		r.Log.Error(err, "unable to find a host for unhealthy machine")
		return ctrl.Result{}, errors.Wrapf(err, "unable to find a host for unhealthy machine")
	}

	// If user has set bmh.Spec.Online to false
	// do not try to remediate the host
	if !remediationMgr.OnlineStatus(host) {
		r.Log.Info("Unable to remediate, Host is powered off (spec.Online is false)")
		remediationMgr.SetRemediationPhase(infrav1.PhaseFailed)
		return ctrl.Result{}, nil
	}

	remediationType := remediationMgr.GetRemediationType()

	if remediationType != infrav1.RebootRemediationStrategy {
		r.Log.Info("unsupported remediation strategy")
		return ctrl.Result{}, nil
	}

	if remediationType == infrav1.RebootRemediationStrategy {
		// If no phase set, default to running and set time and retry count
		if remediationMgr.GetRemediationPhase() == "" {
			remediationMgr.SetRemediationPhase(infrav1.PhaseRunning)
			now := metav1.Now()
			remediationMgr.SetLastRemediationTime(&now)
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		}

		// handle old clusters which were not setup with RBAC for accessing nodes
		isNodeForbidden := false
		node, err := remediationMgr.GetNode(ctx)
		if err != nil {
			if apierrors.IsForbidden(err) {
				r.Log.Info("Node access is forbidden, will skip node deletion")
				isNodeForbidden = true
			} else if !apierrors.IsNotFound(err) {
				r.Log.Error(err, "error getting node for remediation")
				return ctrl.Result{}, errors.Wrap(err, "error getting node for remediation")
			}
		}

		switch remediationMgr.GetRemediationPhase() {
		case infrav1.PhaseRunning:

			return r.remediateRebootStrategy(ctx, remediationMgr, node)

		case infrav1.PhaseWaiting:

			// Node is deleted: remove power off annotation
			ok, err := remediationMgr.IsPowerOffRequested(ctx)
			if err != nil {
				r.Log.Error(err, "error getting poweroff annotation status")
				return ctrl.Result{}, errors.Wrap(err, "error getting poweroff annotation status")
			} else if ok {
				r.Log.Info("Powering on the host")
				err := remediationMgr.RemovePowerOffAnnotation(ctx)
				if err != nil {
					r.Log.Error(err, "error removing poweroff annotation")
					return ctrl.Result{}, errors.Wrap(err, "error removing poweroff annotation")
				}
			}

			// Wait until powered on
			if on, err := remediationMgr.IsPoweredOn(ctx); err != nil {
				r.Log.Error(err, "error getting power status")
				return ctrl.Result{}, errors.Wrap(err, "error getting power status")
			} else if !on {
				// wait a bit before checking again if we are powered on
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			// Restore node if available and not done yet
			if remediationMgr.HasFinalizer() {
				if node != nil {
					// Node was recreated, restore annotations and labels
					r.Log.Info("Restoring the node")
					if err := r.restoreNode(ctx, remediationMgr, node); err != nil {
						return ctrl.Result{}, err
					}

					// clean up
					r.Log.Info("Remediation done, cleaning up remediation CR")
					remediationMgr.RemoveNodeBackupAnnotations()
					remediationMgr.UnsetFinalizer()
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				} else if isNodeForbidden {
					// we don't have a node, just remove finalizer
					remediationMgr.UnsetFinalizer()

					r.Log.Info("Skipping node restore, remediation done, CR should be deleted soon")
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				}
			}

			// Check timeout, either node wasn't recreated yet, or CR is not deleted because of still unhealthy node
			timedOut, _ := remediationMgr.TimeToRemediate(remediationMgr.GetTimeout().Duration)
			if !timedOut {
				// Not yet time to retry or stop remediation, requeue
				r.Log.Info("Waiting for node to get healthy and CR being deleted")
				return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
			}

			// Try again if limit not reached
			if remediationMgr.RetryLimitIsSet() && !remediationMgr.HasReachRetryLimit() {
				r.Log.Info("Remediation timed out, will retry")
				remediationMgr.SetRemediationPhase(infrav1.PhaseRunning)
				now := metav1.Now()
				remediationMgr.SetLastRemediationTime(&now)
				remediationMgr.IncreaseRetryCount()
				return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
			}

			r.Log.Info("Remediation timed out and retry limit reached")

			// When machine is still unhealthy after remediation, and it can be re-provisioned, delete the machine.
			// Note: this differs to the upstream metal3 remediation, which sets a condition which is handled by CAPI
			if ok, err := remediationMgr.CanReprovision(ctx); err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "Failed to check if machine can be reprovisoned")
			} else if ok {
				r.Log.Info("Deleting machine")
				if err := remediationMgr.DeleteMachine(ctx); err != nil {
					return ctrl.Result{}, errors.Wrapf(err, "Failed to delete machine")
				}
			} else {
				r.Log.Info("Machine can't be re-provisioned, will not delete it")
			}

			// Remediation failed, so set unhealthy annotation on BMH
			// This prevents BMH to be selected as a host.
			err = remediationMgr.SetUnhealthyAnnotation(ctx)
			if err != nil {
				r.Log.Error(err, "error setting unhealthy annotation")
				return ctrl.Result{}, errors.Wrapf(err, "error setting unhealthy annotation")
			}

			remediationMgr.SetRemediationPhase(infrav1.PhaseDeleting)
			// no requeue, we are done
			return ctrl.Result{}, nil

		case infrav1.PhaseDeleting:
			// nothing to do anymore
			break

		case infrav1.PhaseFailed:
			// nothing to do anymore
			break

		default:
			r.Log.Error(nil, "unknown phase!", "phase", remediationMgr.GetRemediationPhase())
		}
	}
	return ctrl.Result{}, nil
}

// remediateRebootStrategy executes the remediation using the reboot strategy.
// Returns nil, nil when reconcile can continue.
// Return a Result and optionally an error when reconcile should return.
func (r *Metal3RemediationReconciler) remediateRebootStrategy(ctx context.Context,
	remediationMgr baremetal.RemediationManagerInterface,
	node *corev1.Node) (ctrl.Result, error) {
	// add finalizer
	if !remediationMgr.HasFinalizer() {
		remediationMgr.SetFinalizer()
		return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
	}

	// power off if needed
	if ok, err := remediationMgr.IsPowerOffRequested(ctx); err != nil {
		r.Log.Error(err, "error getting poweroff annotation status")
		return ctrl.Result{}, errors.Wrap(err, "error getting poweroff annotation status")
	} else if !ok {
		r.Log.Info("Powering off the host")
		err = remediationMgr.SetPowerOffAnnotation(ctx)
		if err != nil {
			r.Log.Error(err, "error setting poweroff annotation")
			return ctrl.Result{}, errors.Wrap(err, "error setting poweroff annotation")
		}

		// done for now, wait a bit before checking if we are powered off already
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// wait until powered off
	if on, err := remediationMgr.IsPoweredOn(ctx); err != nil {
		r.Log.Error(err, "error getting power status")
		return ctrl.Result{}, errors.Wrap(err, "error getting power status")
	} else if on {
		// wait a bit before checking again if we are powered off already
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// if we have a node, store annotations and labels, and delete it
	if node != nil {
		/*
			Delete the node only after the host is powered off. Otherwise, if we would delete the node
			when the host is powered on, the scheduler would assign the workload to other nodes, with the
			possibility that two instances of the same application are running in parallel. This might result
			in corruption or other issues for applications with singleton requirement. After the host is powered
			off we know for sure that it is safe to re-assign that workload to other nodes.
		*/
		modified := r.backupNode(remediationMgr, node)
		if modified {
			r.Log.Info("Backing up node")
			// save annotations before deleting node
			return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
		}
		r.Log.Info("Deleting node")
		err := remediationMgr.DeleteNode(ctx, node)
		if err != nil {
			r.Log.Error(err, "error deleting node")
			return ctrl.Result{}, errors.Wrap(err, "error deleting node")
		}
		// wait until node is gone
		return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
	}

	// we are done for this phase, switch to waiting for power on and the node restore
	remediationMgr.SetRemediationPhase(infrav1.PhaseWaiting)
	r.Log.Info("Switch to waiting phase for power on and node restore")
	return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
}

func (r *Metal3RemediationReconciler) getMachine(remediationLog logr.Logger, metal3Remediation *infrav1.Metal3Remediation) (*machinev1beta1.Machine, error) {
	// try to get the machine via owner ref
	for _, ownerRef := range metal3Remediation.OwnerReferences {
		if ownerRef.Kind == "Machine" {
			machineKey := client.ObjectKey{
				Name:      ownerRef.Name,
				Namespace: metal3Remediation.Namespace,
			}
			machine := &machinev1beta1.Machine{}
			err := r.Client.Get(context.Background(), machineKey, machine)
			if err == nil {
				return machine, nil
			} else if !apierrors.IsNotFound(err) {
				remediationLog.Error(err, "failed to get machine from remediation CR owner ref",
					"machine name", machineKey.Name, "namespace", machineKey.Namespace)
				return nil, err
			}
		}
	}
	err := fmt.Errorf("no owner ref with kind Machine found")
	remediationLog.Error(err, "")
	return nil, err
}

// Returns whether annotations or labels were set / updated.
func (r *Metal3RemediationReconciler) backupNode(remediationMgr baremetal.RemediationManagerInterface,
	node *corev1.Node) bool {
	marshaledAnnotations, err := marshal(node.Annotations)
	if err != nil {
		r.Log.Error(err, "failed to marshal node annotations", "node", node.Name)
		// if marshal fails we want to continue without blocking on this, as this error
		// not likely to be resolved in the next run
	}

	marshaledLabels, err := marshal(node.Labels)
	if err != nil {
		r.Log.Error(err, "failed to marshal node labels", "node", node.Name)
	}

	return remediationMgr.SetNodeBackupAnnotations(marshaledAnnotations, marshaledLabels)
}

func (r *Metal3RemediationReconciler) restoreNode(ctx context.Context, remediationMgr baremetal.RemediationManagerInterface,
	node *corev1.Node) error {
	annotations, labels := remediationMgr.GetNodeBackupAnnotations()
	if annotations == "" && labels == "" {
		return nil
	}

	// set annotations
	if len(annotations) > 0 {
		nodeAnnotations, err := unmarshal(annotations)
		if err != nil {
			r.Log.Error(err, "failed to unmarshal node annotations", "node", node.Name, "annotations", annotations)
			// if unmarshal fails we want to continue without blocking on this, as this error
			// is not likely to be resolved in the next run
		}
		if len(nodeAnnotations) > 0 {
			node.Annotations = mergeMaps(node.Annotations, nodeAnnotations)
		}
	}

	// set labels
	if len(labels) > 0 {
		nodeLabels, err := unmarshal(labels)
		if err != nil {
			r.Log.Error(err, "failed to unmarshal node labels", "node", node.Name, "labels", labels)
			// if unmarshal fails we want to continue without blocking on this, as this error
			// is not likely to be resolved in the next run
		}
		if len(nodeLabels) > 0 {
			node.Labels = mergeMaps(node.Labels, nodeLabels)
		}
	}

	if err := remediationMgr.UpdateNode(ctx, node); err != nil {
		r.Log.Error(err, "failed to update node", "node", node.Name)
	}

	return nil
}

// marshal is a wrapper for json.marshal() and converts its output to string.
// if m is nil - an empty string will be returned.
func marshal(m map[string]string) (string, error) {
	if m == nil {
		return "", nil
	}

	marshaled, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	return string(marshaled), nil
}

// unmarshal is a wrapper for json.Unmarshal() for marshaled strings that represent map[string]string.
func unmarshal(marshaled string) (map[string]string, error) {
	if marshaled == "" {
		return make(map[string]string), nil
	}

	decodedValue := make(map[string]string)

	if err := json.Unmarshal([]byte(marshaled), &decodedValue); err != nil {
		return nil, err
	}

	return decodedValue, nil
}

// mergeMaps takes entries from mapToMerge and adds them to prioritizedMap, if entry key
// does not already exist in prioritizedMap. It returns the merged map.
func mergeMaps(prioritizedMap map[string]string, mapToMerge map[string]string) map[string]string {
	if prioritizedMap == nil {
		prioritizedMap = make(map[string]string)
	}

	for key, value := range mapToMerge {
		if _, exists := prioritizedMap[key]; !exists {
			prioritizedMap[key] = value
		}
	}

	return prioritizedMap
}

// SetupWithManager will add watches for Metal3Remediation controller.
func (r *Metal3RemediationReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&infrav1.Metal3Remediation{}).
		Complete(r)
}
