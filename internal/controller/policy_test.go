package controller

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/runtime"

	"github.com/google/go-cmp/cmp"

	pomeriumconfig "github.com/pomerium/pomerium/config"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	networkingv1beta1 "k8s.io/api/extensions/v1beta1"
)

func Test_policyFromObj(t *testing.T) {
	tests := []struct {
		name       string
		wantPolicy []pomeriumconfig.Policy
		obj        func() runtime.Object
		fakeObjs   []runtime.Object
		wantErr    bool
	}{
		{
			name: "ingress-http",
			wantPolicy: []pomeriumconfig.Policy{
				{
					From:          "https://test.lan.beyondcorp.org",
					To:            "http://test-service.default.svc.cluster.local:443",
					AllowedGroups: []string{"foo", "bar"},
				},
				{
					From:          "https://test.lan.beyondcorp.org",
					To:            "http://test-service-string.default.svc.cluster.local:443",
					AllowedGroups: []string{"foo", "bar"},
				},
			},
			fakeObjs: []runtime.Object{
				&corev1.Service{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "v1",
						Kind:       "Service",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-service-string",
						Namespace: "default",
						Annotations: map[string]string{
							"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
							"kubernetes.io/service.class":        "pomerium",
						},
					},
					Spec: corev1.ServiceSpec{
						Ports: []corev1.ServicePort{
							{Name: "https", Port: 443},
						},
					},
				},
			},
			obj: func() runtime.Object {
				o := &networkingv1beta1.Ingress{}
				o.ObjectMeta.Name = "test"
				o.Kind = "Ingress"
				o.Namespace = "default"
				o.ObjectMeta.Annotations = map[string]string{
					"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
					"kubernetes.io/ingress.class":        "pomerium",
				}
				o.Spec.Rules = append(o.Spec.Rules,
					networkingv1beta1.IngressRule{
						Host: "test.lan.beyondcorp.org",
						IngressRuleValue: networkingv1beta1.IngressRuleValue{
							HTTP: &networkingv1beta1.HTTPIngressRuleValue{
								Paths: []networkingv1beta1.HTTPIngressPath{
									{
										Backend: networkingv1beta1.IngressBackend{
											ServiceName: "test-service",
											ServicePort: intstr.FromInt(443),
										},
									},
									{
										Backend: networkingv1beta1.IngressBackend{
											ServiceName: "test-service-string",
											ServicePort: intstr.FromString("https"),
										},
									},
								},
							},
						},
					},
				)
				o.Spec.Backend = &networkingv1beta1.IngressBackend{
					ServiceName: "default-service",
					ServicePort: intstr.FromInt(443),
				}
				return o
			},
		},
		{
			name: "service-https",
			wantPolicy: []pomeriumconfig.Policy{
				{To: "https://test-service.default.svc.cluster.local:443", From: "https://test.lan.beyondcorp.org", AllowedEmails: []string{"user@beyondcorp.org"}},
				{To: "https://test-service.default.svc.cluster.local:9000", From: "https://test.lan.beyondcorp.org", AllowedEmails: []string{"user@beyondcorp.org"}},
			},
			obj: func() runtime.Object {
				o := &corev1.Service{}
				o.Kind = "Service"
				o.Namespace = "default"
				o.ObjectMeta.Name = "test-service"
				o.ObjectMeta.Annotations = map[string]string{
					"ingress.pomerium.io/allowed_users":               `["user@beyondcorp.org"]`,
					"kubernetes.io/ingress.class":                     "pomerium",
					"pomerium.ingress.kubernetes.io/backend-protocol": "https",
					"ingress.pomerium.io/from":                        "https://test.lan.beyondcorp.org",
				}
				o.Spec.Ports = []corev1.ServicePort{
					{Name: "https", Port: 443},
					{Name: "metrics", Port: 9000},
				}
				return o
			},
		},
		{
			name:       "empty",
			wantPolicy: []pomeriumconfig.Policy{},
			obj: func() runtime.Object {
				return &corev1.Service{}
			},
		},
		{
			name:       "bad type",
			wantPolicy: []pomeriumconfig.Policy{},
			obj: func() runtime.Object {
				return &corev1.ConfigMap{}
			},
			wantErr: true,
		},
		{
			name:       "missing service for string port",
			wantPolicy: []pomeriumconfig.Policy{},
			obj: func() runtime.Object {
				o := &networkingv1beta1.Ingress{}
				o.ObjectMeta.Name = "test"
				o.Kind = "Ingress"
				o.Namespace = "default"
				o.ObjectMeta.Annotations = map[string]string{
					"ingress.pomerium.io/allowed_groups": `["foo","bar"]`,
					"kubernetes.io/ingress.class":        "pomerium",
				}
				o.Spec.Rules = append(o.Spec.Rules,
					networkingv1beta1.IngressRule{
						Host: "test.lan.beyondcorp.org",
						IngressRuleValue: networkingv1beta1.IngressRuleValue{
							HTTP: &networkingv1beta1.HTTPIngressRuleValue{
								Paths: []networkingv1beta1.HTTPIngressPath{
									{
										Backend: networkingv1beta1.IngressBackend{
											ServiceName: "test-service-string",
											ServicePort: intstr.FromString("https"),
										},
									},
								},
							},
						},
					},
				)
				o.Spec.Backend = &networkingv1beta1.IngressBackend{
					ServiceName: "default-service",
					ServicePort: intstr.FromString("https"),
				}
				return o
			},
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			obj := test.obj()
			client := fake.NewFakeClient(test.fakeObjs...)
			rec := &Reconciler{}
			err := rec.InjectClient(client)
			assert.NoError(t, err, "failed to inject client")
			policy, err := rec.policyFromObj(obj)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Empty(t, cmp.Diff(test.wantPolicy, policy))
			}
		})
	}
}
