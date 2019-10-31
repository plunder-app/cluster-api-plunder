module github.com/plunder-app/cluster-api-provider-plunder

go 1.12

require (
	github.com/go-logr/logr v0.1.0
	github.com/onsi/ginkgo v1.8.0
	github.com/onsi/gomega v1.5.0
	github.com/prometheus/common v0.4.1
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	k8s.io/klog v0.4.0
	sigs.k8s.io/cluster-api v0.2.6
	sigs.k8s.io/controller-runtime v0.3.0
)
