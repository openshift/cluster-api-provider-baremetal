# Warning

Master branch of this repo is now deprecated and development has moved to https://github.com/openshift/machine-api-operator.git

This repo will only accept back ports for [4.1](https://github.com/openshift/cluster-api/tree/openshift-4.1-cluster-api-0.0.0-alpha.4) and [4.2](https://github.com/openshift/cluster-api/tree/openshift-4.2-cluster-api-0.1.0) bugfixes.

# Cluster API

Cluster API provides the ability to manage Kubernetes supportable hosts in the
context of OpenShift.

This branch contains an implementation of a machineset-controller and
machine-controller as well as their supporting libraries.

Each of these controllers is deployed by the
[machine-api-operator](https://github.com/openshift/machine-api-operator)

# Upstream Implementation
Other branches of this repository may choose to track the upstream
Kubernetes [Cluster-API project](https://github.com/kubernetes-sigs/cluster-api)

In the future, we may align the master branch with the upstream project as it
stabilizes within the community.
