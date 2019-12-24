package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/travisgroth/pomerium-operator/internal/configmanager"

	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/travisgroth/pomerium-operator/internal/log"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logger = log.L.WithValues("component", "reconciler")

type Reconciler struct {
	client.Client
	controllerAnnotation string
	controllerClass      string
	kind                 runtime.Object
	scheme               *runtime.Scheme
	configManager        *configmanager.ConfigManager
}

func NewReconciler(obj runtime.Object, controllerClass string, configManager *configmanager.ConfigManager) *Reconciler {
	r := &Reconciler{}
	r.kind = obj
	r.controllerClass = controllerClass
	r.scheme = scheme.Scheme
	r.configManager = configManager

	gkv, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		logger.Error(err, "could not determine gkv from object", "object", obj)
		return nil
	}

	kindString := strings.ToLower(gkv.Kind)
	r.controllerAnnotation = fmt.Sprintf("kubernetes.io/%s.class", kindString)
	return r
}

func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	logger.V(1).Info("notified of change to resource", "resource", req.NamespacedName)

	obj := r.newKind()
	objName := req.NamespacedName
	objKind := obj.GetObjectKind()
	resource := configmanager.ResourceIdentifier{
		GVK: objKind.GroupVersionKind(), NamespacedName: objName,
	}

	if err := r.Get(context.Background(), req.NamespacedName, obj); err != nil {
		logger.V(1).Info("resource deleted", "resource", resource)
		r.RemoveRoute(resource)
	} else {
		logger.V(1).Info("resource added or modified", "resource", resource)
		r.UpsertRoute(resource, obj)
	}

	return reconcile.Result{}, nil
}

func (r *Reconciler) UpsertRoute(resource configmanager.ResourceIdentifier, obj runtime.Object) {
	if !r.ControllerClassMatch(obj.(metav1.Object)) {
		logger.V(1).Info("resource does not match controller annotation", "resource", resource)
		return
	}

	policy, err := r.policyFromObj(obj)
	if err != nil {
		logger.Error(err, "could not generate policy from object", "id", resource)
		return
	}

	logger.V(1).Info("got resource with policy", "policy", policy, "resource", resource)
	r.configManager.Set(resource, policy)
}

func (r *Reconciler) RemoveRoute(resource configmanager.ResourceIdentifier) {
	logger.V(1).Info("removing resource", "resource", resource)
	err := r.configManager.Remove(resource)
	if err != nil {
		logger.Error(err, "could not remove resource from configuration", "resource", resource)
	}
}

func (r *Reconciler) newKind() runtime.Object {
	k := reflect.ValueOf(r.kind)
	return k.Interface().(runtime.Object)
}

// ControllerClassMatch determines if an Object matches the controllerClass of the Reconciler
func (r *Reconciler) ControllerClassMatch(meta metav1.Object) bool {
	annotations := meta.GetAnnotations()
	class, exists := annotations[r.controllerAnnotation]
	return exists && class == r.controllerClass
}
