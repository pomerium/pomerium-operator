package operator

import (
	"net/http"

	"github.com/pomerium/pomerium-operator/internal/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var logger = log.L.WithValues("component", "operator")
var zHandler = func(_ *http.Request) error { return nil }

// Options represents the configuration of an Operator.  Used in NewOperator()
type Options struct {
	NameSpace               string
	ServiceClass            string
	IngressClass            string
	ConfigMap               string
	Client                  manager.NewClientFunc
	KubeConfig              *rest.Config
	MapperProvider          func(*rest.Config) (meta.RESTMapper, error)
	MetricsBindAddress      string
	HealthAddress           string
	LeaderElection          bool
	LeaderElectionID        string
	LeaderElectionNamespace string
	StopCh                  <-chan struct{}
}

// Operator is the high level wrapper around a manager and the group of controllers that represents the primary functionality of pomerium-operator.  Use NewOperator() to initialize.
//
// Operator supports multiple Controller/Reconciler instances to allow for multiple object type recinciliation under a single controller-manager.
type Operator struct {
	opts    Options
	mgr     manager.Manager
	builder *builder.Builder
	stopCh  <-chan struct{}
}

// NewOperator returns a new instance of an Operator, configured according to an Options struct.  The operator will have an initialized but empty Manager with no controllers.
//
// You must call Start() on the returned Operator to begin event loops.
func NewOperator(opts Options) (*Operator, error) {

	mgrOptions := manager.Options{
		Namespace:               opts.NameSpace,
		LeaderElection:          opts.LeaderElection,
		LeaderElectionNamespace: opts.LeaderElectionNamespace,
		LeaderElectionID:        opts.LeaderElectionID,
		NewClient:               opts.Client,
		MapperProvider:          opts.MapperProvider,
		MetricsBindAddress:      opts.MetricsBindAddress,
		HealthProbeBindAddress:  opts.HealthAddress,
	}

	logger.V(1).Info("creating manager for operator")
	mgr, err := manager.New(opts.KubeConfig, mgrOptions)

	if err != nil {
		logger.Error(err, "failed to create manager")
		return nil, err
	}
	logger.V(1).Info("manager created")

	err = mgr.AddHealthzCheck("alive", zHandler)
	if err != nil {
		return nil, err
	}

	err = mgr.AddReadyzCheck("alive", zHandler)
	if err != nil {
		return nil, err
	}

	operator := Operator{opts: opts, mgr: mgr, builder: builder.ControllerManagedBy(mgr), stopCh: opts.StopCh}

	return &operator, nil
}

// Start calls Start() on the underlying controller-manager.  This begins the event handling loops on the controllers associated with the Operator instance.
func (o *Operator) Start() error {
	logger.Info("starting manager")

	if o.opts.LeaderElection {
		logger.Info("waiting for leadership")
	}

	var stopCh <-chan struct{}
	if o.stopCh == nil {
		stopCh = signals.SetupSignalHandler()
	} else {
		stopCh = o.stopCh
	}

	if err := o.mgr.Start(stopCh); err != nil {
		logger.Error(err, "could not start manager")
		return err
	}
	return nil
}

// Add ensures a manager.Runnable is started with the rest of the operator
func (o *Operator) Add(f manager.Runnable) error {
	return o.mgr.Add(f)
}

// CreateController registers a new Reconciler with the Operator and associates it with an object type to handle events for.
func (o *Operator) CreateController(reconciler reconcile.Reconciler, name string, object runtime.Object) error {
	log.L.V(1).Info("adding controller", "name", name, "kind", object.GetObjectKind().GroupVersionKind().Kind)
	err := o.builder.For(object).Named(name).Complete(reconciler)
	if err != nil {
		logger.Error(err, "failed to create controller", "name", name, "kind", object.GetObjectKind().GroupVersionKind().String())
		return err
	}

	return nil
}
