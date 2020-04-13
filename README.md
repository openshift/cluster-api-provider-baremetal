# Cluster API Provider for Managed Bare Metal Hardware

This repository contains a Machine actuator implementation for the
Kubernetes [Cluster API](https://github.com/kubernetes-sigs/cluster-api/).

For more information about this actuator and related repositories, see
[metal3.io](http://metal3.io/).

## Development Environment

* See [metal3-dev-env](https://github.com/metal3-io/metal3-dev-env) for an
  end-to-end development and test environment for
  `cluster-api-provider-baremetal` and
  [baremetal-operator](https://github.com/metal3-io/baremetal-operator).
* [Setting up for tests](docs/dev/setup.md)
* Using [Minikube](docs/dev/minikube.md)
* Using [OpenShift 4](docs/dev/openshift.md)

## API

See the [API Documentation](docs/api.md) for details about the `providerSpec`
API used with this `cluster-api` provider.

## Machine Remediation

MachinHealthCheck Controller in [Machine-API operator](https://github.com/openshift/machine-api-operator) is checking Node's health.
If you would like to remediate unhealthy Machines you should add the following
to MachineHealthCheck CRD:
```
annotations:
   machine.openshift.io/remediation-strategy: external-baremetal
```

Remediation is done by power-cycling the Host associated with the unhealthy Machine,
using [BMO reboot API](https://github.com/metal3-io/metal3-docs/blob/master/design/reboot-interface.md)

Remediation steps (triggered by the annotation mentioned above):
1) Power off the host
2) Add poweredOffForRemediation annotation to the unhealthy Machine
3) Delete the node
4) Power on the host
5) Wait for the node the come up (by waiting for the node to be registered in the cluster)
6) Remove poweredOffForRemediation annotation and the MAO's machine unhealthy annotation
