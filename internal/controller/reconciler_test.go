package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	corev1 "k8s.io/api/core/v1"

	networkingv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/pomerium/pomerium-operator/internal/configmanager"
	"github.com/stretchr/testify/assert"
)

func Test_NewReconciler(t *testing.T) {
	tests := []struct {
		name                         string
		obj                          runtime.Object
		class                        string
		expectedPanic                bool
		expectedControllerAnnotation string
	}{
		{
			name:                         "ingress",
			obj:                          &networkingv1beta1.Ingress{},
			class:                        "ingress",
			expectedPanic:                false,
			expectedControllerAnnotation: "kubernetes.io/ingress.class",
		},
		{
			name:                         "service",
			obj:                          &corev1.Service{},
			class:                        "service",
			expectedPanic:                false,
			expectedControllerAnnotation: "kubernetes.io/service.class",
		},
		{
			name:                         "ingress",
			obj:                          &networkingv1beta1.Ingress{},
			class:                        ")(",
			expectedPanic:                true,
			expectedControllerAnnotation: "kubernetes.io/ingress.class",
		},
	}

	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeClient := fake.NewFakeClient()
			if tt.expectedPanic {
				assert.Panics(t, func() {
					NewReconciler(tt.obj, tt.class, configmanager.NewConfigManager("test", "test", fakeClient, time.Nanosecond*1))
				})
				return
			}

			c := NewReconciler(tt.obj, tt.class, configmanager.NewConfigManager("test", "test", fakeClient, time.Nanosecond*1))
			assert.NoError(t, c.InjectClient(fakeClient))
			assert.Equal(t, c.kind, tt.obj)
			assert.NotNil(t, c.controllerClassRegExp)
			assert.Equal(t, c.controllerAnnotation, tt.expectedControllerAnnotation)
		})
	}
}

func Test_Reconcile(t *testing.T) {

	tests := []struct {
		name             string
		reconcilerType   runtime.Object
		reconcileRequest reconcile.Request
	}{
		{
			name:           "add-ingress",
			reconcilerType: &networkingv1beta1.Ingress{},
			reconcileRequest: reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "ingress",
					Namespace: "test",
				},
			},
		},
		// {name: "remove-ingress"},
		// {name: "update-ingress"},
	}
	o := fakeObjects()

	// t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			c := fake.NewFakeClient(o...)
			r := NewReconciler(tt.reconcilerType, "pomerium", configmanager.NewConfigManager("test", "test", c, time.Nanosecond*1))

			err := r.InjectClient(c)
			assert.NoError(t, err, "failed to inject client")

			_, err = r.Reconcile(tt.reconcileRequest)
			assert.NoError(t, err)
			// assert.True(t, false)

		})
	}
	// assert.True(t, false)

}

func fakeObjects() []runtime.Object {
	objs := make([]runtime.Object, 0)

	objs = append(objs,
		&networkingv1beta1.Ingress{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "Ingress",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ignored-ingress",
				Namespace: "test",
			},
		},
	)

	objs = append(objs,
		&networkingv1beta1.Ingress{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "extensions/v1beta1",
				Kind:       "Ingress",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "ingress",
				Namespace: "test",
				Annotations: map[string]string{
					"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
					"kubernetes.io/ingress.class":        "pomerium",
				},
			},
		},
	)

	return objs
}

func Test_Reconcile_2(t *testing.T) {

	testSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}
	tests := []struct {
		name string
		obj  runtime.Object
	}{
		{
			name: "add-ingress",
			obj: &networkingv1beta1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "extensions/v1beta1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "ingress",
					Namespace: "test",
					Annotations: map[string]string{
						"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
						"ingress.pomerium.io/from":           `https://test.lan.beyondcorp.org`,
						"kubernetes.io/ingress.class":        "pomerium",
					},
				},
				Spec: networkingv1beta1.IngressSpec{
					Backend: &networkingv1beta1.IngressBackend{
						ServiceName: "default-service",
						ServicePort: intstr.IntOrString{
							IntVal: 443,
							StrVal: "https",
						},
					},
				},
			},
		},
		{
			name: "add-service",
			obj: &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service",
					Namespace: "test",
					Annotations: map[string]string{
						"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
						"ingress.pomerium.io/from":           `https://test.lan.beyondcorp.org`,
						"kubernetes.io/service.class":        "pomerium",
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{Name: "https", Port: 443},
						{Name: "metrics", Port: 9000},
					},
				},
			},
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objNameSpacedName := types.NamespacedName{
				Namespace: tt.obj.(metav1.Object).GetNamespace(),
				Name:      tt.obj.(metav1.Object).GetName(),
			}

			c := fake.NewFakeClient(&testSecret)
			cm := configmanager.NewConfigManager("test", "test", c, time.Nanosecond*1)
			r := NewReconciler(tt.obj, "pomerium", cm)
			err := r.InjectClient(c)
			assert.NoError(t, err, "failed to inject client")

			rec := &Reconciler{}
			wantPolicies, err := rec.policyFromObj(tt.obj)
			assert.NoError(t, err)

			// Reconcile without object
			_, err = r.Reconcile(reconcile.Request{NamespacedName: objNameSpacedName})
			assert.NoError(t, err)
			err = cm.Save()
			assert.NoError(t, err, "failed to save config")
			savedOpts, err := cm.GetCurrentConfig()
			assert.NoError(t, err)
			assert.Empty(t, savedOpts.Policies)

			// Reconcile with object
			err = c.Create(context.Background(), tt.obj)
			assert.NoError(t, err, "failed to create object")
			_, err = r.Reconcile(reconcile.Request{NamespacedName: objNameSpacedName})
			assert.NoError(t, err)
			err = cm.Save()
			assert.NoError(t, err, "failed to save config")
			savedOpts, err = cm.GetCurrentConfig()
			assert.NoError(t, err)
			assert.Subset(t, savedOpts.Policies, wantPolicies)

			// Reconcile a delete
			err = c.Delete(context.Background(), tt.obj)
			assert.NoError(t, err, "failed to delete object")
			_, err = r.Reconcile(reconcile.Request{NamespacedName: objNameSpacedName})
			assert.NoError(t, err)
			err = cm.Save()
			assert.NoError(t, err, "failed to save config")
			savedOpts, err = cm.GetCurrentConfig()
			assert.NoError(t, err)
			assert.Empty(t, savedOpts.Policies)

		})
	}
}

func Test_Reconciler_ControllerClassMatch(t *testing.T) {
	// Helpers
	buildTestObjWithClassAnnotation := func(typ string, classAnnotation string) runtime.Object {
		annotations := map[string]string{}
		if classAnnotation != "" {
			annotations[fmt.Sprintf("kubernetes.io/%s.class", typ)] = classAnnotation
		}

		switch typ {
		case "ingress":
			return &networkingv1beta1.Ingress{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "extensions/v1beta1",
					Kind:       "Ingress",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-ingress",
					Annotations: annotations,
				},
			}
		case "service":
			return &corev1.Service{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "v1",
					Kind:       "Service",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:        "test-service",
					Annotations: annotations,
				},
			}
		default:
			return nil
		}
	}

	// Test cases
	testCases := []struct {
		name            string
		controllerClass string
		obj             runtime.Object
		expectedMatch   bool
	}{
		{
			name:            "should match ingress with unannotated class",
			controllerClass: "whatever",
			obj:             buildTestObjWithClassAnnotation("ingress", ""),
			expectedMatch:   true,
		},
		{
			name:            "should match service with unannotated class",
			controllerClass: "whatever",
			obj:             buildTestObjWithClassAnnotation("service", ""),
			expectedMatch:   true,
		},
		{
			name:            "should match ingress with equal class",
			controllerClass: "pomerium",
			obj:             buildTestObjWithClassAnnotation("ingress", "pomerium"),
			expectedMatch:   true,
		},
		{
			name:            "should match service with equal class",
			controllerClass: "pomerium",
			obj:             buildTestObjWithClassAnnotation("service", "pomerium"),
			expectedMatch:   true,
		},
		{
			name:            "should not match ingress with unequal class",
			controllerClass: "nginx",
			obj:             buildTestObjWithClassAnnotation("ingress", "pomerium"),
			expectedMatch:   false,
		},
		{
			name:            "should not match service with unequal class",
			controllerClass: "nginx",
			obj:             buildTestObjWithClassAnnotation("service", "pomerium"),
			expectedMatch:   false,
		},
		{
			name:            "should match ingress with matching pattern class",
			controllerClass: "(nginx|pomerium)",
			obj:             buildTestObjWithClassAnnotation("ingress", "pomerium"),
			expectedMatch:   true,
		},
		{
			name:            "should match service with matching pattern class",
			controllerClass: "(nginx|pomerium)",
			obj:             buildTestObjWithClassAnnotation("service", "pomerium"),
			expectedMatch:   true,
		},
		{
			name:            "should not match ingress with non-matching pattern class",
			controllerClass: "(nginx|pomerium)",
			obj:             buildTestObjWithClassAnnotation("ingress", "traefik"),
			expectedMatch:   false,
		},
		{
			name:            "should not match service with non-matching pattern class",
			controllerClass: "(nginx|pomerium)",
			obj:             buildTestObjWithClassAnnotation("service", "traefik"),
			expectedMatch:   false,
		},
	}

	t.Parallel()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := NewReconciler(tc.obj, tc.controllerClass, configmanager.NewConfigManager("", "test", fake.NewFakeClient(), time.Nanosecond*1))

			actualMatch := r.ControllerClassMatch(tc.obj.(metav1.Object))
			assert.Equal(t, tc.expectedMatch, actualMatch)
		})
	}
}
