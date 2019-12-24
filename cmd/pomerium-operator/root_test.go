package main

import (
	"io/ioutil"
	"os"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"testing"
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
	defer testEnv.Stop()
	testCfg := testEnv.Config

	o, err := createOperator(testCfg)

	assert.NoError(t, err)
	assert.NotNil(t, o)

	cm, _ := newConfigManager(testCfg)
	serviceController(o, cm)
	ingressController(o, cm)

}

func Test_newConfigManager(t *testing.T) {
	testEnv := newTestEnv(t)
	defer testEnv.Stop()

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
			name:       "good",
			kcfg:       testEnv.Config,
			baseConfig: []byte("metrics_address: :9090"),
			wantErr:    false,
		},
		{
			name:    "bad base config",
			kcfg:    testEnv.Config,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpBaseConfigFile, err := ioutil.TempFile("", "configManagerTest.yaml")
			assert.NoError(t, err)
			defer os.Remove(tmpBaseConfigFile.Name())

			if tt.baseConfig != nil {
				tmpBaseConfigFile.Write(tt.baseConfig)
				tmpBaseConfigFile.Sync()
				baseConfigFile = tmpBaseConfigFile.Name()
			}

			cm, err := newConfigManager(tt.kcfg)
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

	kcfgFile.WriteString(emptyConfig)
	kcfg, err := getConfig()
	assert.NoError(t, err)
	assert.NotNil(t, kcfg)
}
