module github.com/openshift/cluster-api-provider-baremetal

go 1.13

require (
	github.com/coreos/ignition/v2 v2.7.0
	github.com/go-logr/zapr v0.2.0 // indirect
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/metal3-io/baremetal-operator v0.0.0-00010101000000-000000000000
	github.com/onsi/gomega v1.10.1
	github.com/openshift/machine-api-operator v0.2.1-0.20201203125141-79567cb3368e
	github.com/openshift/machine-config-operator v0.0.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.6.1
	github.com/vincent-petithory/dataurl v0.0.0-20160330182126-9a301d65acbb
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	k8s.io/api v0.19.2
	k8s.io/apimachinery v0.19.2
	k8s.io/client-go v12.0.0+incompatible
	k8s.io/utils v0.0.0-20200729134348-d5654de09c73
	sigs.k8s.io/controller-runtime v0.6.2
	sigs.k8s.io/controller-tools v0.3.0
	sigs.k8s.io/yaml v1.2.0
)

replace (
	github.com/metal3-io/baremetal-operator => github.com/openshift/baremetal-operator v0.0.0-20200715132148-0f91f62a41fe // Use OpenShift fork
	github.com/openshift/machine-config-operator => github.com/openshift/machine-config-operator v0.0.1-0.20201206051445-34d1cf050f52
	k8s.io/client-go => k8s.io/client-go v0.19.0
	sigs.k8s.io/cluster-api-provider-aws => github.com/openshift/cluster-api-provider-aws v0.2.1-0.20201125052318-b85a18cbf338
	sigs.k8s.io/cluster-api-provider-azure => github.com/openshift/cluster-api-provider-azure v0.1.0-alpha.3.0.20201130182513-88b90230f2a4
)
