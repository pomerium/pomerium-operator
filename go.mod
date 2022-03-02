module github.com/pomerium/pomerium-operator

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v1.1.0
	github.com/go-logr/zapr v1.1.0
	github.com/google/go-cmp v0.5.6
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/googleapis/gnostic v0.5.4 // indirect
	github.com/iancoleman/strcase v0.2.0
	github.com/pomerium/pomerium v0.16.4
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.10.1
	github.com/stretchr/testify v1.7.0
	go.uber.org/zap v1.19.1
	golang.org/x/term v0.0.0-20210615171337-6886f2dfbf5b // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.20.6
	k8s.io/apiextensions-apiserver v0.19.9 // indirect
	k8s.io/apimachinery v0.20.6
	k8s.io/client-go v0.20.6
	k8s.io/klog/v2 v2.8.0 // indirect
	k8s.io/kube-openapi v0.0.0-20210323165736-1a6458611d18 // indirect
	k8s.io/utils v0.0.0-20210305010621-2afb4311ab10 // indirect
	sigs.k8s.io/controller-runtime v0.7.2
	sigs.k8s.io/structured-merge-diff/v4 v4.1.2 // indirect
)
