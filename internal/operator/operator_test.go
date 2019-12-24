package operator

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/stretchr/testify/assert"
	networkingv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/meta/testrestmapper"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func newFakeClient(cache cache.Cache, config *rest.Config, options client.Options) (client.Client, error) {
	return fake.NewFakeClientWithScheme(options.Scheme), nil
}
func newFakeRestMapper(c *rest.Config) (meta.RESTMapper, error) {
	return testrestmapper.TestOnlyStaticRESTMapper(scheme.Scheme), nil
}

type fakeReconciler struct{}

func (f *fakeReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	return reconcile.Result{}, nil
}

func Test_NewOperator(t *testing.T) {

	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             newFakeClient,
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
		Client:    newFakeClient,
	})
	assert.Error(t, err)
}

func Test_CreateController(t *testing.T) {

	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             newFakeClient,
		KubeConfig:         &rest.Config{},
		MapperProvider:     newFakeRestMapper,
		MetricsBindAddress: "0",
	})

	assert.NoError(t, err)

	tests := []struct {
		name string
		obj  runtime.Object
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

func Test_StartController(t *testing.T) {

	stopCh := make(chan struct{})
	o, err := NewOperator(Options{
		NameSpace:          "test",
		Client:             newFakeClient,
		KubeConfig:         &rest.Config{},
		MapperProvider:     newFakeRestMapper,
		MetricsBindAddress: "0",
		StopCh:             stopCh,
	})
	if !assert.NoError(t, err) {
		return
	}

	go func() {
		err = o.Start()
		assert.NoError(t, err)
	}()

	close(stopCh)
}
