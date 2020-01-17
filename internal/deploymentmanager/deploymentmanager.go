package deploymentmanager

import (
	"context"

	"github.com/pomerium/pomerium-operator/internal/log"
	pomeriumconfig "github.com/pomerium/pomerium/config"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logger = log.L.WithValues("component", "deploymentmanager")

const deploymentConfigAnnotation = "pomerium.io/config-checksum"

// Deployment Manager is responsible for maintaining a set of kubernetes Deployments
type DeploymentManager struct {
	namespace   string
	deployments []string
	client      client.Client
}

// NewDeploymentManager returns a new DeploymentManager configured to manage a given set of deployments with a
// kubernetes client
func NewDeploymentManager(deployments []string, namespace string, client client.Client) *DeploymentManager {
	return &DeploymentManager{
		namespace:   namespace,
		deployments: deployments,
		client:      client,
	}
}

// UpdateDeployments implements a ConfigReceiver.  It stores a checksum of the baseBytes as an annotation
// on the managed deployments.  This forces them to update the corresponding ReplicaSet if there are changes
func (d *DeploymentManager) UpdateDeployments(config pomeriumconfig.Options) {

	// Policy is dynamic - clear it
	config.Policies = make([]pomeriumconfig.Policy, 0)

	checksum := config.Checksum()

	logger.V(1).Info("received deployment update", "checksum", checksum)

	for _, name := range d.deployments {

		deploymentName := types.NamespacedName{
			Namespace: d.namespace,
			Name:      name,
		}
		deploymentObj := &appsv1.Deployment{}

		err := d.client.Get(context.Background(), deploymentName, deploymentObj)
		if err != nil {
			logger.Error(err, "failed to retrieve deployment", "deployment", deploymentName.String())
			continue
		}

		if deploymentObj.Spec.Template.Annotations == nil {
			deploymentObj.Spec.Template.Annotations = make(map[string]string)
		}

		deploymentObj.Spec.Template.Annotations[deploymentConfigAnnotation] = checksum

		logger.V(1).Info("updating deployment", "checksum", checksum, "deployment", name)
		err = d.client.Update(context.Background(), deploymentObj)
		if err != nil {
			logger.Error(err, "failed to update deployment", "deployment", deploymentName.String())
			continue
		}
		logger.Info("updated deployment", "checksum", checksum, "deployment", name)
	}
}
