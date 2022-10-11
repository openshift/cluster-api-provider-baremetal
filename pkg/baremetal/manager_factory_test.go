/*
Copyright 2021 The Kubernetes Authors.

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
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Manager factory testing", func() {
	var fakeClient client.Client
	var managerFactory ManagerFactory
	clusterLog := logr.Discard()

	BeforeEach(func() {
		fakeClient = fake.NewClientBuilder().WithScheme(setupScheme()).Build()
		managerFactory = NewManagerFactory(fakeClient)
	})

	It("returns a manager factory", func() {
		Expect(managerFactory.client).To(Equal(fakeClient))
	})

	It("returns a Remediation manager", func() {
		_, err := managerFactory.NewRemediationManager(&infrav1.Metal3Remediation{}, &machinev1beta1.Machine{}, clusterLog)
		Expect(err).NotTo(HaveOccurred())
	})
})
