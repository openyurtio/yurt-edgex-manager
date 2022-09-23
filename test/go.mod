module github.com/openyurtio/yurt-edgex-manager/test

go 1.16

replace github.com/openyurtio/yurt-edgex-manager => ../

require (
	github.com/Microsoft/go-winio v0.5.2 // indirect
	github.com/docker/distribution v2.8.1+incompatible // indirect
	github.com/docker/docker v20.10.16+incompatible
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.19.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/openyurtio/api v0.0.0-20220907024010-e5bfc9cc1b4b
	github.com/openyurtio/yurt-edgex-manager v0.0.0-00010101000000-000000000000
	github.com/pkg/errors v0.9.1
	k8s.io/api v0.24.1
	k8s.io/apiextensions-apiserver v0.24.1
	k8s.io/apimachinery v0.24.1
	k8s.io/client-go v0.24.1
	sigs.k8s.io/cluster-api/test/framework v0.0.0-20200304170348-97097699f713
	sigs.k8s.io/controller-runtime v0.12.1
	sigs.k8s.io/kind v0.14.0
	sigs.k8s.io/yaml v1.3.0
)
