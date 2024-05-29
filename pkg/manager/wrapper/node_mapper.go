package wrapper

import (
	"context"
	"log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const MachineAnnotation = "machine.openshift.io/machine"

// Map will return a reconcile request for a Machine if the event is for a
// Node and that Node references a Machine.
func nodeMap(_ context.Context, obj client.Object) []reconcile.Request {
	if node, ok := obj.(*corev1.Node); ok {
		machineKey, ok := node.Annotations[MachineAnnotation]
		if !ok {
			return []reconcile.Request{}
		}

		namespace, machineName, err := cache.SplitMetaNamespaceKey(machineKey)
		if err != nil {
			log.Printf("Error mapping Node %s to Machine: %s", node.Name, err.Error())
			return []reconcile.Request{}
		}

		return []reconcile.Request{
			reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: namespace,
					Name:      machineName,
				},
			},
		}
	}
	return []reconcile.Request{}
}
