package operator

import (
	"context"
	"sync"
	"testing"

	corev1 "k8s.io/api/core/v1"

	"github.com/stretchr/testify/assert"
	networkingv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type fakeClientBuilder struct {
	objs []client.Object
}

func (f *fakeClientBuilder) Build(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
	return fake.NewClientBuilder().WithObjects(f.objs...).Build(), nil
}

func (f *fakeClientBuilder) WithUncached(objs ...client.Object) manager.ClientBuilder {
	f.objs = objs
	return f
}

var clientBuilder = &fakeClientBuilder{}

func newFakeRestMapper(c *rest.Config) (meta.RESTMapper, error) {
	return testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme), nil
}

type fakeReconciler struct{}

func (f *fakeReconciler) Reconcile(_ context.Context, _ reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func Test_NewOperator(t *testing.T) {

	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             clientBuilder,
		KubeConfig:         &rest.Config{},
		MapperProvider:     newFakeRestMapper,
		MetricsBindAddress: "0",
	})
	if !assert.NoError(t, err) {
		return
	}

	assert.Equal(t, "test", o.opts.NameSpace)
	assert.NoError(t, o.mgr.GetClient().List(context.Background(), &corev1.ServiceList{}))

	_, err = NewOperator(Options{
		NameSpace: "test",
		Client:    clientBuilder,
	})
	assert.Error(t, err)
}

func Test_CreateController(t *testing.T) {

	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             clientBuilder,
		KubeConfig:         &rest.Config{},
		MapperProvider:     newFakeRestMapper,
		MetricsBindAddress: "0",
	})

	assert.NoError(t, err)

	tests := []struct {
		name string
		obj  client.Object
		rec  reconcile.Reconciler
	}{
		{name: "test-ingress", obj: &networkingv1beta1.Ingress{}, rec: &fakeReconciler{}},
		{name: "test-service", obj: &corev1.Service{}, rec: &fakeReconciler{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := o.CreateController(tt.rec, tt.name, tt.obj)
			assert.NoError(t, err)
		})
	}
}

type fakeRunnable struct {
	startCalled bool
	stopCalled  bool
	wg          *sync.WaitGroup
}

func (f *fakeRunnable) Start(ctx context.Context) error {
	defer f.wg.Done()
	f.startCalled = true
	<-ctx.Done()
	f.stopCalled = true
	return nil
}

func Test_StartController(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             clientBuilder,
		KubeConfig:         &rest.Config{},
		MapperProvider:     newFakeRestMapper,
		MetricsBindAddress: "0",
	})
	if !assert.NoError(t, err) {
		return
	}

	wg := &sync.WaitGroup{}

	r := &fakeRunnable{wg: wg}
	wg.Add(1)
	o.Add(r) //nolint: errcheck

	wg.Add(1)
	go func() {
		defer wg.Done()

		err = o.Start(ctx)
		assert.NoError(t, err)
	}()

	cancel()
	wg.Wait()

	// Make sure runnable was stopped
	assert.True(t, r.stopCalled)
	assert.True(t, r.startCalled)
}
