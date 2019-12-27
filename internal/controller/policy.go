package controller

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"k8s.io/apimachinery/pkg/util/intstr"

	"k8s.io/apimachinery/pkg/types"

	gyaml "github.com/ghodss/yaml"
	"github.com/pomerium/pomerium-operator/internal/configmanager"
	yaml "gopkg.in/yaml.v2"

	corev1 "k8s.io/api/core/v1"

	pomeriumconfig "github.com/pomerium/pomerium/config"
	networkingv1beta1 "k8s.io/api/extensions/v1beta1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// policyFromObj returns a pomerium []Policy, mapping the pomerium policy
// parameters onto each Policy implied by the collection of Backends or the
// Service described by obj
//
// In practice you will get a Policy element for each Service referred to
// by obj.  All annotations on obj are attached to each Policy element.
func (r *Reconciler) policyFromObj(obj runtime.Object) ([]pomeriumconfig.Policy, error) {

	metaObj, ok := obj.(metav1.Object)
	if !ok {
		return nil, fmt.Errorf("passed an object which is not of type meta/v1/Object")
	}

	annotations := metaObj.GetAnnotations()

	scheme, ok := annotations["pomerium.ingress.kubernetes.io/backend-protocol"]
	if !ok {
		scheme = "http"
	}
	scheme = strings.ToLower(scheme)

	policyOptions := make(map[string]string)

	for k, v := range annotations {
		// Filter to only the pomerium ingress prefix
		if strings.HasPrefix(k, "ingress.pomerium.io/") {

			policyKey := strings.SplitN(k, "/", 2)[1]
			valueBytes := []byte(v)

			// Support yaml or json in the annotation
			valueJSON, err := gyaml.YAMLToJSON(valueBytes)

			if err != nil {
				policyOptions[policyKey] = v
			} else {
				policyOptions[policyKey] = string(valueJSON)
			}

		}
	}

	// coerce an actual JSON structure from the escaped value in the annotation
	policyOptionsUnescaped := make([]string, 0)
	for k, v := range policyOptions {
		policyOptionsUnescaped = append(policyOptionsUnescaped, fmt.Sprintf("\"%s\": %s", k, v))
	}
	policyOptionsJSON := "{" + strings.Join(policyOptionsUnescaped, ",") + "}"

	policies, err := r.policyHostnamesFromObj(obj.(runtime.Object), scheme)
	if err != nil {
		return nil, err
	}

	// merge settings from annotations onto each policy
	for k := range policies {
		if err := yaml.Unmarshal([]byte(policyOptionsJSON), &policies[k]); err != nil {
			return nil, fmt.Errorf("failed to insert policy options into policy: %w", err)
		}
	}
	return policies, nil
}

// policyHostnamesFromObj returns an array of pomerium policies with the `to` and `from` values mapped
// from the underlying kubernetes data.
//
// In practice, this returns an element for each service port or an element for every host rule + backend on an Ingress.
func (r *Reconciler) policyHostnamesFromObj(obj runtime.Object, scheme string) (policies []pomeriumconfig.Policy, err error) {
	policies = make([]pomeriumconfig.Policy, 0)
	resource, err := configmanager.NewResourceIdentifierFromObj(obj.(metav1.Object))
	if err != nil {
		return policies, err
	}

	switch kind := obj.(type) {
	case *corev1.Service:
		for _, port := range kind.Spec.Ports {
			to := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", scheme, resource.NamespacedName.Name, resource.NamespacedName.Namespace, port.Port)
			policies = append(policies, pomeriumconfig.Policy{To: to})
		}
	case *networkingv1beta1.Ingress:
		for _, rule := range kind.Spec.Rules {
			from := fmt.Sprintf("https://%s", rule.Host)
			for _, path := range rule.HTTP.Paths {
				// to := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", scheme, path.Backend.ServiceName, resource.NamespacedName.Namespace, path.Backend.ServicePort.IntValue())
				backendURL, err := r.backendToURL(path.Backend, resource.NamespacedName.Namespace)
				if err != nil {
					return policies, fmt.Errorf("failed to form DNS for rule '%s' backend: %w", rule.Host, err)
				}

				backendURL.Scheme = scheme
				policies = append(policies, pomeriumconfig.Policy{To: backendURL.String(), From: from})
			}
		}
		if kind.Spec.Backend != nil {
			// to := fmt.Sprintf("%s://%s.%s.svc.cluster.local:%d", scheme, kind.Spec.Backend.ServiceName, resource.NamespacedName.Namespace, kind.Spec.Backend.ServicePort.IntValue())
			backendURL, err := r.backendToURL(*kind.Spec.Backend, resource.NamespacedName.Namespace)
			if err != nil {
				return policies, fmt.Errorf("failed to form DNS for backend: %w", err)
			}

			backendURL.Scheme = scheme
			policies = append(policies, pomeriumconfig.Policy{To: backendURL.String()})
		}
	default:
		return policies, fmt.Errorf("received an incompatible object kind: %s", kind.GetObjectKind().GroupVersionKind().String())
	}
	return policies, nil
}

// backendToURL converts an IngressBackend for a given namespace into a url.URL
func (r *Reconciler) backendToURL(backend networkingv1beta1.IngressBackend, namespace string) (serviceDNS url.URL, err error) {

	var portNum int32
	switch portType := backend.ServicePort.Type; portType {
	case intstr.Int:
		portNum = int32(backend.ServicePort.IntValue())
	case intstr.String:
		serviceRef := types.NamespacedName{Name: backend.ServiceName, Namespace: namespace}
		portNum, err = r.portFromService(serviceRef, backend.ServicePort.String())

		if err != nil {
			return serviceDNS, fmt.Errorf("could not convert string ServicePort to integer: %w", err)
		}
	}

	serviceDNS.Host = fmt.Sprintf("%s.%s.svc.cluster.local:%d", backend.ServiceName, namespace, portNum)

	return serviceDNS, nil
}

// portFromService translates a string based port on a Service into a numeric port
func (r *Reconciler) portFromService(service types.NamespacedName, port string) (int32, error) {
	serviceObj := &corev1.Service{}
	if err := r.Get(context.Background(), service, serviceObj); err != nil {
		return 0, err
	}

	for _, servicePort := range serviceObj.Spec.Ports {
		if servicePort.Name == port {
			return servicePort.Port, nil
		}
	}

	return 0, fmt.Errorf("could not find port on service %s", service)
}
