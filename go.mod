module github.com/openshift/cluster-api-provider-baremetal

go 1.13

require (
	github.com/metal3-io/baremetal-operator v0.0.0
	github.com/onsi/gomega v1.9.0
	github.com/openshift/machine-api-operator v0.2.1-0.20200825143304-b66cbe4106dd
	github.com/stretchr/testify v1.4.0
	golang.org/x/net v0.0.0-20200501053045-e0ff5e5a1de5
	k8s.io/api v0.18.2
	k8s.io/apimachinery v0.18.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/utils v0.0.0-20200414100711-2df71ebbae66 // indirect
	sigs.k8s.io/controller-runtime v0.6.0
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // Use OpenShift fork
	k8s.io/client-go => k8s.io/client-go v0.18.2
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20200520125206-5e266b553d8e
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20200529030741-17d4edc5142f
	sigs.k8s.io/controller-runtime => sigs.k8s.io/controller-runtime v0.5.1-0.20200330174416-a11a908d91e0
)
