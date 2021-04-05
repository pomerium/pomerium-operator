package controller

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/pomerium/pomerium-operator/internal/configmanager"

	"k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/pomerium/pomerium-operator/internal/log"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logger = log.L.WithValues("component", "reconciler")

// Reconciler implements a Kubernetes reconciler for either a Service or Ingress resources.  Use NewReconciler() to initialize.
type Reconciler struct {
	client.Client
	controllerAnnotation  string
	controllerClass       string
	controllerClassRegExp *regexp.Regexp
	kind                  runtime.Object
	scheme                *runtime.Scheme
	configManager         *configmanager.ConfigManager
}

// NewReconciler returns a new Reconciler for obj type Objects.
//
// configManager is called with configuration updates from reconcile cycles.
//
// controllerClass filters resources based on matching `kubernetes.io/XXXXX.class` where XXXXX is based on obj's type.
func NewReconciler(obj runtime.Object, controllerClass string, configManager *configmanager.ConfigManager) *Reconciler {
	r := &Reconciler{}
	r.kind = obj
	r.scheme = scheme.Scheme
	r.configManager = configManager
	r.controllerClass = controllerClass
	if strings.HasPrefix(controllerClass, "/") && strings.HasSuffix(controllerClass, "/") {
		r.controllerClassRegExp = regexp.MustCompile(controllerClass[1 : len(controllerClass)-1])
	}

	gkv, err := apiutil.GVKForObject(obj, scheme.Scheme)
	if err != nil {
		logger.Error(err, "could not determine gkv from object", "object", obj)
		return nil
	}

	kindString := strings.ToLower(gkv.Kind)
	r.controllerAnnotation = fmt.Sprintf("kubernetes.io/%s.class", kindString)
	return r
}

// InjectClient implements the Reconciler interface and accepts a new initialized client to be used by the reconciler internally
func (r *Reconciler) InjectClient(c client.Client) error {
	r.Client = c
	return nil
}

// Reconcile implements the Reconciler interface and conducts a reconcile loop on a given request.  This is typically called by a controller-manager like that found inside an Operator.
func (r *Reconciler) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	logger.V(1).Info("notified of change to resource", "resource", req.NamespacedName)

	obj := r.newKind()
	objName := req.NamespacedName
	objKind := obj.GetObjectKind()
	resource := configmanager.ResourceIdentifier{
		GVK: objKind.GroupVersionKind(), NamespacedName: objName,
	}

	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		logger.V(1).Info("resource deleted", "resource", resource)
		r.RemoveRoute(resource)
	} else {
		logger.V(1).Info("resource added or modified", "resource", resource)
		r.UpsertRoute(resource, obj)
	}

	return reconcile.Result{}, nil
}

// UpsertRoute adds or updates a route entry into the ConfigManager associated with this Reconciler.  It will only do so if the controllerClass matches or is absent
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

	if len(policy) == 0 {
		logger.V(1).Info("no policy generated", "resource", resource)
		return
	}

	logger.V(1).Info("got resource with policy", "policy", policy, "resource", resource)
	r.configManager.Set(resource, policy)
}

// RemoveRoute removes a route entry from the ConfigManager associated with this Reconciler, if it currently exists.
//
// It is not an error to remove a route which is not present.
func (r *Reconciler) RemoveRoute(resource configmanager.ResourceIdentifier) {
	logger.V(1).Info("removing resource", "resource", resource)
	err := r.configManager.Remove(resource)
	if err != nil {
		logger.Error(err, "could not remove resource from configuration", "resource", resource)
	}
}

func (r *Reconciler) newKind() client.Object {
	k := reflect.ValueOf(r.kind)
	return k.Interface().(client.Object)
}

// ControllerClassMatch determines if an Object matches the controllerClass of the Reconciler or has no controllerClass
func (r *Reconciler) ControllerClassMatch(meta metav1.Object) bool {
	annotations := meta.GetAnnotations()
	class, exists := annotations[r.controllerAnnotation]
	if !exists {
		return true
	}

	if r.controllerClassRegExp != nil {
		return r.controllerClassRegExp.MatchString(class)
	}

	return r.controllerClass == class
}
