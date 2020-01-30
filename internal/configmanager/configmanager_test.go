package configmanager

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pomeriumconfig "github.com/pomerium/pomerium/config"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var mockBaseConfig = pomeriumconfig.Options{
	InsecureServer:       true,
	ForwardAuthURLString: "https://nginx-hates-you.beyondcorp.org",
	Policies:             []pomeriumconfig.Policy{},
}

func mockBaseConfigBytes(t *testing.T) []byte {
	bytes, err := yaml.Marshal(mockBaseConfig)
	if !assert.NoError(t, err) {
		t.Fatal()
	}

	return bytes
}

func newMockClient(t *testing.T) client.Client {

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pomerium",
			Namespace: "test",
		},
		Data: make(map[string]string),
	}
	client := fake.NewFakeClient(configMap)
	return client
}

func newIngressResourceIdentifier(name string) ResourceIdentifier {
	return ResourceIdentifier{
		GVK: schema.GroupVersionKind{
			Group:   "networking",
			Version: "v1beta1",
			Kind:    "Ingress",
		},
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: "test",
		},
	}
}

func Test_getBaseConfig(t *testing.T) {
	c := newMockClient(t)
	cm := NewConfigManager("test", "pomerium", c, time.Nanosecond*1)
	err := cm.SetBaseConfig(mockBaseConfigBytes(t))
	assert.NoError(t, err, "could not set base config")

	opt, err := cm.getBaseConfig()

	assert.Empty(t, cmp.Diff(
		cm.baseConfig,
		mockBaseConfigBytes(t),
	))
	assert.Empty(t, cmp.Diff(opt, mockBaseConfig, cmpopts.IgnoreUnexported(pomeriumconfig.Options{})))
	assert.NoError(t, err)
}

func Test_Save(t *testing.T) {

	policyList := map[ResourceIdentifier][]pomeriumconfig.Policy{

		newIngressResourceIdentifier("test-ingress"): []pomeriumconfig.Policy{
			{To: "https://to-test.beyondcorp.local", From: "https://from-test.beyondcorp.local"},
			{To: "https://to-test2.beyondcorp.local", From: "https://from-test2.beyondcorp.local"},
		},
		newIngressResourceIdentifier("deleted-ingress"): []pomeriumconfig.Policy{
			{To: "https://to-deleted.beyondcorp.local", From: "https://from-deleted.beyondcorp.local"},
			{To: "https://to-deleted2.beyondcorp.local", From: "https://from-deleted2.beyondcorp.local"},
		},
		newIngressResourceIdentifier("overridden-ingress"): []pomeriumconfig.Policy{
			{To: "https://to-overridden.beyondcorp.local", From: "https://from-overridden.beyondcorp.local"},
			{To: "https://to-overridden2.beyondcorp.local", From: "https://from-overridden2.beyondcorp.local"},
		},
		newIngressResourceIdentifier("override-ingress"): []pomeriumconfig.Policy{
			{To: "https://to-override.beyondcorp.local", From: "https://from-override.beyondcorp.local"},
			{To: "https://to-override2.beyondcorp.local", From: "https://from-override2.beyondcorp.local"},
		},
	}

	set := []struct {
		name   string
		id     ResourceIdentifier
		policy []pomeriumconfig.Policy
	}{
		{
			name:   "normal policy",
			id:     newIngressResourceIdentifier("test-ingress"),
			policy: policyList[newIngressResourceIdentifier("test-ingress")],
		},
		{
			name:   "deleted policy",
			id:     newIngressResourceIdentifier("deleted-ingress"),
			policy: policyList[newIngressResourceIdentifier("deleted-ingress")],
		},
		{
			name:   "overridden policy",
			id:     newIngressResourceIdentifier("overridden-ingress"),
			policy: policyList[newIngressResourceIdentifier("overridden-ingress")],
		},
		{
			name:   "override policy",
			id:     newIngressResourceIdentifier("overridden-ingress"),
			policy: policyList[newIngressResourceIdentifier("override-ingress")],
		},
	}

	remove := []struct {
		name    string
		id      ResourceIdentifier
		wantErr bool
	}{
		{
			name:    "normal",
			id:      newIngressResourceIdentifier("deleted-ingress"),
			wantErr: false,
		},
		{
			name:    "missing",
			id:      newIngressResourceIdentifier("absent-ingress"),
			wantErr: true,
		},
	}

	client := newMockClient(t)
	t.Parallel()
	cm := NewConfigManager("test", "pomerium", client, time.Nanosecond*1)
	err := cm.SetBaseConfig(mockBaseConfigBytes(t))
	assert.NoError(t, err, "could not set base config")

	for _, tt := range set {
		t.Run(tt.name, func(t *testing.T) {
			cm.Set(tt.id, tt.policy)
			err := cm.Save()
			assert.NoError(t, err, "failed to save")
		})
	}

	for _, tt := range remove {
		t.Run(tt.name, func(t *testing.T) {
			err := cm.Remove(tt.id)
			assert.Equal(t, (err != nil), tt.wantErr)
			err = cm.Save()
			assert.NoError(t, err, "failed to save")
		})
	}

	configMap := &corev1.ConfigMap{}
	err = client.Get(context.Background(), types.NamespacedName{Name: cm.configMap, Namespace: cm.namespace}, configMap)
	assert.NoError(t, err, "failed to get configMap")

	resultOptions := pomeriumconfig.Options{}
	configBytes := []byte(configMap.Data[configKey])
	err = yaml.Unmarshal(configBytes, &resultOptions)
	assert.NoError(t, err, "failed to unmarshal resulting config")

	assert.Empty(t,
		cmp.Diff(
			resultOptions,
			mockBaseConfig,
			cmpopts.IgnoreUnexported(pomeriumconfig.Options{}),
			cmpopts.IgnoreFields(pomeriumconfig.Options{}, "Policies"),
		),
		"found difference between unmarshalled and desired pomerium options",
	)

	persistedConfig, err := cm.GetPersistedConfig()
	assert.NoError(t, err)

	currentConfig, err := cm.GetCurrentConfig()
	assert.NoError(t, err)

	assert.Empty(t,
		cmp.Diff(
			resultOptions,
			persistedConfig,
			cmpopts.IgnoreUnexported(pomeriumconfig.Options{}),
		),
		"found difference between unmarshalled and stored pomerium options",
	)

	assert.Empty(t,
		cmp.Diff(
			persistedConfig,
			currentConfig,
			cmpopts.IgnoreUnexported(pomeriumconfig.Options{}),
			cmpopts.IgnoreFields(pomeriumconfig.Options{}, "Policies"),
		),
		"found difference between persisted and current config",
	)
	assert.Subset(t, persistedConfig.Policies, currentConfig.Policies, "found difference between persisted and current config")

	assert.Subset(t, resultOptions.Policies, policyList[newIngressResourceIdentifier("test-ingress")])
	assert.Subset(t, resultOptions.Policies, policyList[newIngressResourceIdentifier("override-ingress")])
	assert.NotSubset(t, resultOptions.Policies, policyList[newIngressResourceIdentifier("deleted-ingress")])
	assert.NotSubset(t, resultOptions.Policies, policyList[newIngressResourceIdentifier("overridden-ingress")])

}

func Test_Save_clientError(t *testing.T) {

	cm := NewConfigManager("test", "pomerium", fake.NewFakeClient(), time.Nanosecond*1)
	assert.Error(t, cm.Save())

}

func Test_Save_unmarshalError(t *testing.T) {
	garbage := []byte("not,yaml!")
	cm := NewConfigManager("test", "pomerium", newMockClient(t), time.Nanosecond*1)

	assert.Error(t, cm.SetBaseConfig(garbage))

	cm.baseConfig = garbage
	assert.Error(t, cm.Save())

}

func Test_SaveLoop(t *testing.T) {
	cm := NewConfigManager("test", "pomerium", newMockClient(t), time.Nanosecond*1)
	cm.Set(newIngressResourceIdentifier("test"), []pomeriumconfig.Policy{{To: "foo", From: "bar"}})

	stopCh := make(chan struct{})
	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		cm.Start(stopCh) //nolint: errcheck
	}()

	close(stopCh)
	wg.Wait()
	persistedOpts, err := cm.GetPersistedConfig()
	assert.NoError(t, err)
	assert.NotEmpty(t, persistedOpts)

}

type mockSaveCallback struct {
	called       int
	calledConfig pomeriumconfig.Options
}

func (m *mockSaveCallback) Call(config pomeriumconfig.Options) {
	m.called++
	m.calledConfig = config
}

func Test_OnSave(t *testing.T) {
	callback := &mockSaveCallback{}

	baseConfig := mockBaseConfigBytes(t)
	cm := NewConfigManager("test", "pomerium", newMockClient(t), time.Nanosecond*1)

	err := cm.SetBaseConfig(baseConfig)
	cm.Set(newIngressResourceIdentifier("test"), []pomeriumconfig.Policy{{To: "foo", From: "bar"}})
	assert.NoError(t, err)

	cm.OnSave(callback.Call)

	err = cm.Save()
	assert.NoError(t, err)
	persistedConfig, err := cm.GetPersistedConfig()
	assert.NoError(t, err)

	assert.Equal(t, 1, callback.called)
	assert.Empty(t, cmp.Diff(persistedConfig, callback.calledConfig, cmpopts.IgnoreUnexported(pomeriumconfig.Options{})))

	cm.OnSave(callback.Call)
	err = cm.Save()

	assert.NoError(t, err)
	assert.Equal(t, 3, callback.called)
	assert.Empty(t, cmp.Diff(persistedConfig, callback.calledConfig, cmpopts.IgnoreUnexported(pomeriumconfig.Options{})))
}
