package wrapper

import (
	"fmt"
	"log"

	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// New returns a new manager wrapper. It intercepts the controller when it gets
// added and causes that controller to Watch BareMetalHost objects.
func New(mgr manager.Manager) manager.Manager {
	return &managerWrapper{
		Manager: mgr,
	}
}

// managerWrapper is a wrapper around a "real manager". It intercepts the
// Controller when it gets added and causes that controller to Watch
// BareMetalHost objects.
type managerWrapper struct {
	manager.Manager
}

// Add causes the controller to Watch for BareMetalHost events, and then calls
// the wrapped manager's Add function.
func (m *managerWrapper) Add(r manager.Runnable) error {
	err := m.Manager.Add(r)
	if err != nil {
		return err
	}

	c, ok := r.(controller.Controller)
	if !ok {
		return fmt.Errorf("Runnable was not a Controller")
	}

	if c == nil {
		return fmt.Errorf("Controller was nil")
	}

	err = c.Watch(source.Kind(m.GetCache(), &bmh.BareMetalHost{}), handler.EnqueueRequestsFromMapFunc(bmhMap))
	if err != nil {
		log.Printf("Error watching BareMetalHosts: %s", err.Error())
		return err
	}

	err = c.Watch(source.Kind(m.GetCache(), &corev1.Node{}), handler.EnqueueRequestsFromMapFunc(nodeMap))
	if err != nil {
		log.Printf("Error watching Nodes: %s", err.Error())
		return err
	}

	return nil
}
