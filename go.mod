module github.com/pomerium/pomerium-operator

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-redis/redis v6.15.6+incompatible // indirect
	github.com/google/go-cmp v0.4.1
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/pomerium/go-oidc v2.0.0+incompatible // indirect
	github.com/pomerium/pomerium v0.7.5
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.6.3
	github.com/stretchr/testify v1.5.1
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.16.6
	sigs.k8s.io/controller-runtime v0.4.0
)
