module github.com/openyurtio/yurt-edgex-manager

go 1.16

require (
	github.com/google/uuid v1.2.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/openyurtio/api v0.0.0-20220907024010-e5bfc9cc1b4b
	github.com/pkg/errors v0.9.1
	gopkg.in/check.v1 v1.0.0-20201130134442-10cb98267c6c // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/api v0.24.1
	k8s.io/apimachinery v0.24.1
	k8s.io/client-go v0.24.1
	k8s.io/utils v0.0.0-20220210201930-3a6ce19ff2f9
	sigs.k8s.io/cluster-api v1.1.3
	sigs.k8s.io/controller-runtime v0.12.1
)
