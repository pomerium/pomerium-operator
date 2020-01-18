package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/pomerium/pomerium-operator/internal/configmanager"
	"github.com/pomerium/pomerium-operator/internal/controller"
	"github.com/pomerium/pomerium-operator/internal/deploymentmanager"
	"github.com/pomerium/pomerium-operator/internal/log"
	"github.com/pomerium/pomerium-operator/internal/operator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	kubeConfig          string
	debug               bool
	namespace           string
	serviceClass        string
	ingressClass        string
	pomeriumNamespace   string
	pomeriumConfigMap   string
	electionConfigMap   string
	electionNamespace   string
	electionEnabled     bool
	metricsAddress      string
	baseConfigFile      string
	logger              = log.L
	pomeriumDeployments []string
)

var rootCmd = &cobra.Command{
	Use:   "pomerium-operator",
	Short: "pomerium-operator is a kubernetes operator for pomerium identity aware proxy",
	RunE: func(cmd *cobra.Command, args []string) error {
		if debug {
			log.Debug()
		}

		kcfg, err := getConfig()
		if err != nil {
			return err
		}

		o, err := createOperator(kcfg)
		if err != nil {
			return err
		}

		kClient, err := newRestClient(kcfg)
		if err != nil {
			return err
		}

		configManager, err := newConfigManager(kClient)
		if err != nil {
			return err
		}

		deploymentManager := deploymentmanager.NewDeploymentManager(kClient, pomeriumDeployments, pomeriumNamespace)
		configManager.OnSave(deploymentManager.UpdateDeployments)

		if err := ingressController(o, configManager); err != nil {
			return err
		}
		if err := serviceController(o, configManager); err != nil {
			return err
		}

		go configManager.Start()

		if err = o.Start(); err != nil {
			logger.Error(err, "operator failed to start.  exiting")
			return err
		}

		return nil
	},
}

func main() {
	err := viper.BindPFlags(rootCmd.PersistentFlags())
	if err != nil {
		fmt.Println(fmt.Errorf("failed to bind pflags: %w", err))
		os.Exit(1)
	}
	viper.AutomaticEnv()

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
func init() {
	rootCmd.PersistentFlags().StringVar(&kubeConfig, "kubeconfig", "", "Path to kubeconfig file")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Run in debug mode")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Namespaces to monitor")
	rootCmd.PersistentFlags().StringVar(&pomeriumConfigMap, "pomerium-configmap", "pomerium", "Name of pomerium ConfigMap to maintain")
	rootCmd.PersistentFlags().StringVar(&pomeriumNamespace, "pomerium-namespace", "kube-system", "Name of pomerium ConfigMap to maintain")
	rootCmd.PersistentFlags().StringVar(&baseConfigFile, "base-config-file", "./pomerium-base.yaml", "Path to base configuration file")

	rootCmd.PersistentFlags().StringVarP(&serviceClass, "service-class", "s", "pomerium", "kubernetes.io/service.class to monitor")
	rootCmd.PersistentFlags().StringVarP(&ingressClass, "ingress-class", "i", "pomerium", "kubernetes.io/ingress.class to monitor")

	rootCmd.PersistentFlags().BoolVar(&electionEnabled, "election", false, "Enable leader election (for running multiple controller replicas)")
	rootCmd.PersistentFlags().StringVar(&electionConfigMap, "election-configmap", "operator-leader-pomerium", "Name of ConfigMap to use for leader election")
	rootCmd.PersistentFlags().StringVar(&electionNamespace, "election-namespace", "kube-system", "Namespace to use for leader election")
	rootCmd.PersistentFlags().StringVar(&metricsAddress, "metrics-address", "0", "Address for metrics listener.  Default disabled")
	rootCmd.PersistentFlags().StringSliceVar(&pomeriumDeployments, "pomerium-deployments", []string{}, "List of Deployments in the pomerium-namespace to update when the [base-config-file] changes")

}

func newRestClient(config *rest.Config) (client.Client, error) {
	c, err := client.New(config, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client for config manager: %w", err)
	}

	return c, nil
}

func newConfigManager(kClient client.Client) (cm *configmanager.ConfigManager, err error) {
	cm = configmanager.NewConfigManager(pomeriumNamespace, pomeriumConfigMap, kClient, time.Second*10)

	baseBytes, err := ioutil.ReadFile(baseConfigFile)
	if err != nil {

		return cm, fmt.Errorf("failed to load base config file: %w", err)
	}

	if err := cm.SetBaseConfig(baseBytes); err != nil {
		return cm, fmt.Errorf("failed to set base config from %s: %w", baseConfigFile, err)
	}
	return
}

func ingressReconciler(cm *configmanager.ConfigManager) *controller.Reconciler {
	ingressResource := &extensionsv1beta1.Ingress{}
	return controller.NewReconciler(ingressResource, ingressClass, cm)
}

func serviceReconciler(cm *configmanager.ConfigManager) *controller.Reconciler {
	serviceResource := &corev1.Service{}
	return controller.NewReconciler(serviceResource, serviceClass, cm)
}

func ingressController(o *operator.Operator, cm *configmanager.ConfigManager) (err error) {
	ingressResource := &extensionsv1beta1.Ingress{}
	reconciler := ingressReconciler(cm)

	if err := o.CreateController(reconciler, "pomerium-ingress", ingressResource); err != nil {
		return fmt.Errorf("could not register ingress controller: %w", err)
	}

	return nil
}

func serviceController(o *operator.Operator, cm *configmanager.ConfigManager) (err error) {
	serviceResource := &corev1.Service{}
	reconciler := serviceReconciler(cm)

	if err := o.CreateController(reconciler, "pomerium-service", serviceResource); err != nil {
		return fmt.Errorf("could not register service controller: %w", err)

	}

	return nil
}

func createOperator(kcfg *rest.Config) (*operator.Operator, error) {
	o, err := operator.NewOperator(
		operator.Options{
			KubeConfig:         kcfg,
			NameSpace:          namespace,
			ServiceClass:       serviceClass,
			IngressClass:       ingressClass,
			MetricsBindAddress: metricsAddress,
		},
	)
	return o, err
}

func getConfig() (*rest.Config, error) {
	logger.V(1).Info("loading kubeconfig")
	kcfg, err := config.GetConfig()
	if err != nil {
		logger.Error(err, "failed to find kubeconfig")
		return nil, err
	}

	logger.V(1).Info("found kubeconfig.  connecting.", "api-server", kcfg.Host)
	return kcfg, nil
}
