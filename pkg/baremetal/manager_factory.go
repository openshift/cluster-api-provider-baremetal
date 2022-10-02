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

	infrav1 "github.com/metal3-io/cluster-api-provider-metal3/api/v1beta1"
	machinev1beta1 "github.com/openshift/api/machine/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ManagerFactoryInterface is a collection of new managers.
type ManagerFactoryInterface interface {
	NewRemediationManager(*infrav1.Metal3Remediation, *machinev1beta1.Machine, logr.Logger) (
		RemediationManagerInterface, error,
	)
}

// ManagerFactory only contains a client.
type ManagerFactory struct {
	client client.Client
}

// NewManagerFactory returns a new factory.
func NewManagerFactory(client client.Client) ManagerFactory {
	return ManagerFactory{client: client}
}

// NewRemediationManager creates a new RemediationManager.
func (f ManagerFactory) NewRemediationManager(remediation *infrav1.Metal3Remediation,
	ocpMachine *machinev1beta1.Machine,
	remediationLog logr.Logger) (RemediationManagerInterface, error) {
	return NewRemediationManager(f.client, remediation, ocpMachine, remediationLog)
}
