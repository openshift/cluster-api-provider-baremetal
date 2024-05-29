package wrapper

import (
	"context"
	"testing"

	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestMap(t *testing.T) {
	var obj client.Object

	for _, tc := range []struct {
		Host          *bmh.BareMetalHost
		ExpectRequest bool
	}{
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "myns",
				},
				Spec: bmh.BareMetalHostSpec{
					ConsumerRef: &corev1.ObjectReference{
						Name:       "someothermachine",
						Namespace:  "myns",
						Kind:       "Machine",
						APIVersion: machinev1beta1.SchemeGroupVersion.String(),
					},
				},
			},
			ExpectRequest: true,
		},
		{
			Host: &bmh.BareMetalHost{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "host1",
					Namespace: "myns",
				},
				Spec: bmh.BareMetalHostSpec{},
			},
			ExpectRequest: false,
		},
	} {
		obj = tc.Host

		reqs := bmhMap(context.TODO(), obj)

		if tc.ExpectRequest {
			if len(reqs) != 1 {
				t.Errorf("Expected 1 request, found %d", len(reqs))
			}
			req := reqs[0]
			if req.NamespacedName.Name != tc.Host.Spec.ConsumerRef.Name {
				t.Errorf("Expected name %s, found %s", tc.Host.Spec.ConsumerRef.Name, req.NamespacedName.Name)
			}
			if req.NamespacedName.Namespace != tc.Host.Spec.ConsumerRef.Namespace {
				t.Errorf("Expected namespace %s, found %s", tc.Host.Spec.ConsumerRef.Namespace, req.NamespacedName.Namespace)
			}

		} else {
			if len(reqs) != 0 {
				t.Errorf("Expected 0 request, found %d", len(reqs))
			}
		}
	}
}
