module github.com/plunder-app/cluster-api-plunder

go 1.12

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	github.com/plunder-app/cluster-api-plunder v0.0.0-20191103114358-3413f694fd37
	//github.com/plunder-app/plunder v0.4.5 // indirect
	github.com/plunder-app/plunder/pkg/apiserver v0.0.0
	github.com/plunder-app/plunder/pkg/parlay/parlaytypes v0.0.0-00010101000000-000000000000
	github.com/plunder-app/plunder/pkg/plunderlogging v0.0.0-00010101000000-000000000000
	github.com/plunder-app/plunder/pkg/services v0.0.0-00010101000000-000000000000
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/spf13/cobra v0.0.5 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/cluster-bootstrap v0.0.0-20190516232516-d7d78ab2cfe7 // indirect
	k8s.io/klog v1.0.0
	sigs.k8s.io/cluster-api v0.2.6
	sigs.k8s.io/controller-runtime v0.3.0
	sigs.k8s.io/controller-tools v0.2.0-beta.4 // indirect
)

replace (
	github.com/plunder-app/plunder/pkg/apiserver => ../plunder/pkg/apiserver
	github.com/plunder-app/plunder/pkg/parlay/parlaytypes => ../plunder/pkg/parlay/parlaytypes
	github.com/plunder-app/plunder/pkg/plunderlogging => ../plunder/pkg/plunderlogging
	github.com/plunder-app/plunder/pkg/services => ../plunder/pkg/services
	github.com/plunder-app/plunder/pkg/utils => ../plunder/pkg/utils

)
