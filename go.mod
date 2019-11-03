module github.com/plunder-app/cluster-api-provider-plunder

go 1.12

require (
	github.com/c4milo/gotoolkit v0.0.0-20190525173301-67483a18c17a // indirect
	github.com/go-logr/logr v0.1.0
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/hooklift/assert v0.0.0-20170704181755-9d1defd6d214 // indirect
	github.com/hooklift/iso9660 v1.0.0 // indirect
	github.com/krolaw/dhcp4 v0.0.0-20190909130307-a50d88189771 // indirect
	//github.com/plunder-app/plunder v0.4.5 // indirect
	github.com/plunder-app/plunder/pkg/apiserver v0.0.0
	github.com/plunder-app/plunder/pkg/parlay/parlaytypes v0.0.0-00010101000000-000000000000
	github.com/plunder-app/plunder/pkg/plunderlogging v0.0.0-00010101000000-000000000000
	github.com/plunder-app/plunder/pkg/services v0.0.0-00010101000000-000000000000
	github.com/plunder-app/plunder/pkg/utils v0.0.0-00010101000000-000000000000
	github.com/prometheus/common v0.4.1
	github.com/thebsdbox/go-tftp v0.0.0-20190329154032-a7263f18c49c // indirect
	github.com/whyrusleeping/go-tftp v0.0.0-20180830013254-3695fa5761ee // indirect
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	k8s.io/klog v0.4.0
	sigs.k8s.io/cluster-api v0.2.6
	sigs.k8s.io/controller-runtime v0.3.0
)

replace (
	github.com/plunder-app/plunder/pkg/apiserver => ../plunder/pkg/apiserver
	github.com/plunder-app/plunder/pkg/parlay/parlaytypes => ../plunder/pkg/parlay/parlaytypes
	github.com/plunder-app/plunder/pkg/plunderlogging => ../plunder/pkg/plunderlogging
	github.com/plunder-app/plunder/pkg/services => ../plunder/pkg/services
	github.com/plunder-app/plunder/pkg/utils => ../plunder/pkg/utils

)
