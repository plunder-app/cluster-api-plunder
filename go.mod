module github.com/plunder-app/cluster-api-plunder

go 1.12

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/c4milo/gotoolkit v0.0.0-20190525173301-67483a18c17a // indirect
	github.com/go-logr/logr v0.1.0
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/hashicorp/golang-lru v0.5.3 // indirect
	github.com/hooklift/assert v0.0.0-20170704181755-9d1defd6d214 // indirect
	github.com/hooklift/iso9660 v1.0.0 // indirect
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/json-iterator/go v1.1.7 // indirect
	github.com/krolaw/dhcp4 v0.0.0-20190909130307-a50d88189771 // indirect
	github.com/onsi/ginkgo v1.10.1 // indirect
	github.com/onsi/gomega v1.7.0 // indirect
	//github.com/plunder-app/cluster-api-plunder v0.0.0-20191103114358-3413f694fd37
	// 	github.com/plunder-app/plunder v0.4.5 // indirect
	github.com/plunder-app/plunder/pkg/apiserver v0.0.0-20191105152536-b5c505aaf830
	github.com/plunder-app/plunder/pkg/parlay/parlaytypes v0.0.0-20191105152536-b5c505aaf830
	github.com/plunder-app/plunder/pkg/plunderlogging v0.0.0-20191105152536-b5c505aaf830
	github.com/plunder-app/plunder/pkg/services v0.0.0-20191105152536-b5c505aaf830
	github.com/plunder-app/plunder/pkg/utils v0.0.0-20191105152536-b5c505aaf830
	github.com/prometheus/procfs v0.0.5 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/thebsdbox/go-tftp v0.0.0-20190329154032-a7263f18c49c // indirect
	github.com/whyrusleeping/go-tftp v0.0.0-20180830013254-3695fa5761ee // indirect
	golang.org/x/oauth2 v0.0.0-20190402181905-9f3314589c9a // indirect
	google.golang.org/appengine v1.5.0 // indirect
	gopkg.in/yaml.v2 v2.2.4 // indirect
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v11.0.1-0.20190409021438-1a26190bd76a+incompatible
	k8s.io/klog v1.0.0
	sigs.k8s.io/cluster-api v0.2.7
	sigs.k8s.io/controller-runtime v0.3.0
)

replace (
	k8s.io/api => k8s.io/api v0.0.0-20190704095032-f4ca3d3bdf1d
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20190704094733-8f6ac2502e51
	sigs.k8s.io/cluster-api => sigs.k8s.io/cluster-api v0.0.0-20190829144357-1063658f9b58

)
