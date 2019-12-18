package operator

import (
	"github.com/travisgroth/pomerium-operator/internal/log"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var logger = log.L.WithValues("component", "operator")

type OperatorOptions struct {
	NameSpace               string
	ServiceClass            string
	IngressClass            string
	ConfigMap               string
	Client                  manager.NewClientFunc
	KubeConfig              *rest.Config
	MapperProvider          func(*rest.Config) (meta.RESTMapper, error)
	MetricsBindAddress      string
	LeaderElection          bool
	LeaderElectionID        string
	LeaderElectionNamespace string
	StopCh                  <-chan struct{}
}

type Operator struct {
	opts    OperatorOptions
	mgr     manager.Manager
	builder *builder.Builder
	stopCh  <-chan struct{}
}

func NewOperator(opts OperatorOptions) (*Operator, error) {

	mgrOptions := manager.Options{
		Namespace:               opts.NameSpace,
		LeaderElection:          opts.LeaderElection,
		LeaderElectionNamespace: opts.LeaderElectionNamespace,
		LeaderElectionID:        opts.LeaderElectionID,
		NewClient:               opts.Client,
		MapperProvider:          opts.MapperProvider,
		MetricsBindAddress:      opts.MetricsBindAddress,
	}

	logger.V(1).Info("creating manager for operator")
	mgr, err := manager.New(opts.KubeConfig, mgrOptions)

	if err != nil {
		logger.Error(err, "failed to create manager")
		return nil, err
	}
	logger.V(1).Info("manager created")

	operator := Operator{opts: opts, mgr: mgr, builder: builder.ControllerManagedBy(mgr)}

	return &operator, nil
}

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

func (o *Operator) CreateController(reconciler reconcile.Reconciler, name string, object runtime.Object) error {
	log.L.V(1).Info("adding controller", "name", name, "kind", object.GetObjectKind().GroupVersionKind().Kind)
	err := o.builder.For(object).Named(name).Complete(reconciler)
	if err != nil {
		logger.Error(err, "failed to create controller", "name", name, "kind", object.GetObjectKind().GroupVersionKind().String())
		return err
	}

	return nil
}
