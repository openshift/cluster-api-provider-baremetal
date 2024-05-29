package wrapper

import (
	"context"

	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// Map will return a reconcile request for a Machine if the event is for a
// BareMetalHost and that BareMetalHost references a Machine.
func bmhMap(_ context.Context, obj client.Object) []reconcile.Request {
	if host, ok := obj.(*bmh.BareMetalHost); ok {
		if host.Spec.ConsumerRef != nil && host.Spec.ConsumerRef.Kind == "Machine" && host.Spec.ConsumerRef.APIVersion == machinev1beta1.SchemeGroupVersion.String() {
			return []reconcile.Request{
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      host.Spec.ConsumerRef.Name,
						Namespace: host.Spec.ConsumerRef.Namespace,
					},
				},
			}
		}
	}
	return []reconcile.Request{}
}
