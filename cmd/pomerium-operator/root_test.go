package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/pomerium/pomerium-operator/internal/deploymentmanager"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

func newTestEnv(t *testing.T) *envtest.Environment {
	t.Helper()
	os.Setenv("KUBEBUILDER_ASSETS", "../../.kubebuilder/bin")
	env := &envtest.Environment{}
	_, err := env.Start()
	if !assert.NoError(t, err) {
		t.Fatalf("Could not start control plane: %s", err)
	}

	return env
}

func Test_main_setup(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.Stop() //nolint:errcheck
	testCfg := testEnv.Config

	o, err := createOperator(testCfg)

	assert.NoError(t, err)
	assert.NotNil(t, o)

	kClient, _ := newRestClient(testCfg)
	cm, _ := newConfigManager(kClient)
	dm := deploymentmanager.NewDeploymentManager(kClient, []string{"pomerium-proxy"}, "test")
	cm.OnSave(dm.UpdateDeployments)
	err = serviceController(o, cm)
	assert.NoError(t, err, "could not create service controller")

	err = ingressController(o, cm)
	assert.NoError(t, err, "could not create ingress controller")

}

func Test_newRestClient(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.Stop() //nolint: errcheck

	tests := []struct {
		name       string
		kcfg       *rest.Config
		baseConfig []byte
		wantErr    bool
	}{
		{
			name:    "no config",
			kcfg:    nil,
			wantErr: true,
		},
		{
			name:    "good",
			kcfg:    testEnv.Config,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := newRestClient(tt.kcfg)
			assert.Equal(t, tt.wantErr, err != nil)

			if tt.wantErr {
				return
			}

			assert.NotNil(t, c)
			assert.NoError(t, err)
		})
	}
}

func Test_newConfigManager(t *testing.T) {
	tests := []struct {
		name       string
		baseConfig []byte
		wantErr    bool
	}{
		{
			name:       "good",
			baseConfig: []byte("metrics_address: :9090"),
			wantErr:    false,
		},
		{
			name:    "bad base config",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpBaseConfigFile, err := ioutil.TempFile("", "configManagerTest.yaml")
			assert.NoError(t, err)
			defer os.Remove(tmpBaseConfigFile.Name())

			if tt.baseConfig != nil {

				_, err := tmpBaseConfigFile.Write(tt.baseConfig)
				assert.NoError(t, err, "could not create base configuration file")

				err = tmpBaseConfigFile.Close()
				assert.NoError(t, err, "could not close base configuration file")

				operatorCfg.BaseConfigFile = tmpBaseConfigFile.Name()
			}

			kClient := fake.NewFakeClient()
			cm, err := newConfigManager(kClient)
			assert.Equal(t, tt.wantErr, err != nil)

			if tt.wantErr {
				return
			}

			assert.NotNil(t, cm)
			currentConfig, err := cm.GetCurrentConfig()
			assert.NoError(t, err)
			assert.NotEmpty(t, currentConfig)
		})
	}

}

func Test_getConfig(t *testing.T) {

	kcfgFile, err := ioutil.TempFile("", "pomerium-operator_test-kube-config.yaml")
	if !assert.NoError(t, err) {
		assert.FailNow(t, "could not generate temp file: %w", err)
	}

	defer os.Remove(kcfgFile.Name())
	defer os.Unsetenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kcfgFile.Name())

	_, err = getConfig()
	assert.Error(t, err)

	var emptyConfig = `
apiVersion: v1
clusters:
- cluster:
    server: https://1.2.3.4
  name: test
contexts:
- context:
    cluster: test
    user: ""
  name: test
current-context: test
kind: Config
preferences: {}
users: []`

	_, err = kcfgFile.WriteString(emptyConfig)
	assert.NoError(t, err, "could not write out kube config file")
	kcfg, err := getConfig()
	assert.NoError(t, err)
	assert.NotNil(t, kcfg)
}

func Test_bindViper(t *testing.T) {

	testFlagValue := "Test Flag"
	testEnvValue := "Test Env"
	tests := []struct {
		name   string
		flag   string
		env    string
		option string
	}{
		{"simple", "namespace", "NAMESPACE", "Namespace"},
		{"kebab", "base-config-file", "BASE_CONFIG_FILE", "BaseConfigFile"},
		{"boolean", "debug", "DEBUG", "Debug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer os.Unsetenv(tt.env)

			v := viper.New()
			flagSet := &pflag.FlagSet{}
			flagSet.String(tt.flag, "", "")

			err := bindViper(v, flagSet)
			assert.NoError(t, err)

			os.Setenv(tt.env, testEnvValue)
			assert.Equal(t, testEnvValue, v.GetString(tt.option))

			err = flagSet.Set(tt.flag, testFlagValue)
			assert.NoError(t, err)
			assert.Equal(t, testFlagValue, v.GetString(tt.option))
		})
	}

}
