package configmanager

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

// A ResourceIdentifier is a Map-compatible representation of a cluster-unique name of a resource.  It captures Group, Version, Kind, Namespace and Name of the resource.
type ResourceIdentifier struct {
	GVK            schema.GroupVersionKind
	NamespacedName types.NamespacedName
}

// NewResourceIdentifierFromObj returns a new ResourceIdentifier derived from the attributes of the obj passed in
func NewResourceIdentifierFromObj(obj metav1.Object) (ResourceIdentifier, error) {
	namespacedName := types.NamespacedName{
		Name:      obj.GetName(),
		Namespace: obj.GetNamespace(),
	}

	r := ResourceIdentifier{}

	kinds, _, err := scheme.Scheme.ObjectKinds(obj.(runtime.Object))
	if err != nil {
		return r, fmt.Errorf("could not lookup object in schema: %w", err)
	}
	r = ResourceIdentifier{
		GVK:            kinds[0],
		NamespacedName: namespacedName,
	}

	return r, nil
}
