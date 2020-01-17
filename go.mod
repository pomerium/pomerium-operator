module github.com/pomerium/pomerium-operator

go 1.13

require (
	github.com/apache/thrift v0.12.0 // indirect
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.0
	github.com/google/go-cmp v0.4.0
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/openzipkin/zipkin-go v0.1.6 // indirect
	github.com/pomerium/pomerium v0.5.1-0.20200114220137-2f6142eb354e
	github.com/spf13/cobra v0.0.5
	github.com/spf13/viper v1.6.1
	github.com/stretchr/testify v1.4.0
	go.uber.org/zap v1.10.0
	gopkg.in/yaml.v2 v2.2.7
	k8s.io/api v0.0.0-20190918195907-bd6ac527cfd2
	k8s.io/apimachinery v0.0.0-20190817020851-f2f3a405f61d
	k8s.io/client-go v0.0.0-20190918200256-06eb1244587a
	sigs.k8s.io/controller-runtime v0.3.0
)
