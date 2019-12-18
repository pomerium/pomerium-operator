package configmanager

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
)

type ResourceIdentifier struct {
	GVK            schema.GroupVersionKind
	NamespacedName types.NamespacedName
}

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
