module github.com/pomerium/pomerium-operator

go 1.14

require (
	github.com/ghodss/yaml v1.0.0
	github.com/go-logr/logr v0.1.0
	github.com/go-logr/zapr v0.1.1
	github.com/go-redis/redis/v7 v7.3.0 // indirect
	github.com/google/go-cmp v0.5.1
	github.com/google/gofuzz v1.1.0 // indirect
	github.com/iancoleman/strcase v0.0.0-20191112232945-16388991a334
	github.com/imdario/mergo v0.3.8 // indirect
	github.com/pomerium/autocache v0.0.0-20200505053831-8c1cd659f055 // indirect
	github.com/pomerium/pomerium v0.10.1-0.20200807184328-fbb367d39320
	github.com/spf13/cobra v1.0.0
	github.com/spf13/pflag v1.0.5
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.6.1
	github.com/zenazn/goji v0.9.0 // indirect
	go.etcd.io/bbolt v1.3.4 // indirect
	go.uber.org/zap v1.15.0
	gopkg.in/yaml.v2 v2.3.0
	k8s.io/api v0.17.3
	k8s.io/apimachinery v0.17.3
	k8s.io/client-go v0.16.6
	sigs.k8s.io/controller-runtime v0.4.0
)
