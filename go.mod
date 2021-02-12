module github.com/openshift/cluster-api-provider-baremetal

go 1.13

require (
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/metal3-io/baremetal-operator v0.0.0-00010101000000-000000000000
	github.com/onsi/gomega v1.10.1
	github.com/openshift/machine-api-operator v0.2.1-0.20201119210302-9643c3accfda
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.5.1
	golang.org/x/net v0.0.0-20200707034311-ab3426394381
	k8s.io/api v0.19.0
	k8s.io/apimachinery v0.19.0
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog/v2 v2.3.0
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // Use OpenShift fork
	k8s.io/client-go => k8s.io/client-go v0.19.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200929152424-eab2e087f366
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200929220456-04e680e51d03
)
