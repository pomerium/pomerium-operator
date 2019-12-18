package configmanager

import (
	"testing"

	networkingv1beta1 "k8s.io/api/extensions/v1beta1"

	"github.com/google/go-cmp/cmp"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
)

func Test_NewResourceIdentifierFromObj(t *testing.T) {

	tests := []struct {
		name           string
		obj            runtime.Object
		wantIdentifier ResourceIdentifier
	}{
		{
			name: "service",
			obj: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-service",
					Namespace: "testing",
				},
			},
			wantIdentifier: ResourceIdentifier{
				GVK: schema.GroupVersionKind{
					Group:   "",
					Version: "v1",
					Kind:    "Service",
				},
				NamespacedName: types.NamespacedName{
					Name:      "test-service",
					Namespace: "testing",
				},
			},
		},
		{
			name: "ingress",
			obj: &networkingv1beta1.Ingress{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-ingress",
					Namespace: "testing",
				},
			},
			wantIdentifier: ResourceIdentifier{
				GVK: schema.GroupVersionKind{
					Group:   "extensions",
					Version: "v1beta1",
					Kind:    "Ingress",
				},
				NamespacedName: types.NamespacedName{
					Name:      "test-ingress",
					Namespace: "testing",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			i, err := NewResourceIdentifierFromObj(tt.obj.(metav1.Object))
			assert.NoError(t, err)
			assert.Empty(t, cmp.Diff(tt.wantIdentifier, i))
		})

	}
}
