package deploymentmanager

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	pomeriumconfig "github.com/pomerium/pomerium/config"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newMockDeployment(name string, namespace string, annotations map[string]string) *appsv1.Deployment {
	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: annotations,
				},
			},
		},
	}

	return d
}

func Test_UpdateDeployments(t *testing.T) {
	managerNamespace := "test"

	tests := []struct {
		name        string
		namespace   string
		annotations map[string]string
		config      pomeriumconfig.Options
		wantErr     bool
	}{
		{
			name:        "add-checksum",
			namespace:   "test",
			config:      pomeriumconfig.Options{ForwardAuthURLString: "https://forward-auth.beyondcorp.org"},
			annotations: make(map[string]string),
		},
		{
			name:      "no-change",
			namespace: "other",
			config:    pomeriumconfig.Options{},
			wantErr:   true,
		},
		{
			name:        "change-only-checksum",
			namespace:   "test",
			config:      pomeriumconfig.Options{ForwardAuthURLString: "https://forward-auth.beyondcorp.org"},
			annotations: map[string]string{"foo": "bar"},
		},
		{
			name:        "update-checksum",
			namespace:   "test",
			config:      pomeriumconfig.Options{ForwardAuthURLString: "https://forward-auth.beyondcorp.org"},
			annotations: map[string]string{deploymentConfigAnnotation: fmt.Sprintf("%x", sha256.Sum256([]byte("baz")))},
		},
	}

	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployment := newMockDeployment(tt.name, tt.namespace, tt.annotations)
			c := fake.NewFakeClient(deployment)

			dm := NewDeploymentManager([]string{tt.name}, managerNamespace, c)
			dm.UpdateDeployments(tt.config)

			updatedDeployment := &appsv1.Deployment{}
			err := c.Get(context.Background(), types.NamespacedName{Name: tt.name, Namespace: tt.namespace}, updatedDeployment)
			assert.NoError(t, err)

			checksummedConfig := tt.config
			checksummedConfig.Policies = make([]pomeriumconfig.Policy, 0)

			wantedAnnotations := tt.annotations

			// wantErr stands in for "no update expected"
			if !tt.wantErr {
				wantedAnnotations[deploymentConfigAnnotation] = checksummedConfig.Checksum()
			}
			assert.Equal(t, wantedAnnotations, updatedDeployment.Spec.Template.Annotations)

		})
	}
}
