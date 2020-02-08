package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/iancoleman/strcase"
	"github.com/pomerium/pomerium-operator/internal/configmanager"
	"github.com/pomerium/pomerium-operator/internal/controller"
	"github.com/pomerium/pomerium-operator/internal/deploymentmanager"
	"github.com/pomerium/pomerium-operator/internal/log"
	"github.com/pomerium/pomerium-operator/internal/operator"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

var (
	vcfg        *viper.Viper = viper.New()
	logger                   = log.L
	operatorCfg              = &cmdConfig{}
)

type cmdConfig struct {
	BaseConfigFile    string
	Debug             bool
	Election          bool
	ElectionConfigMap string
	ElectionNamespace string

	IngressClass        string
	MetricsAddress      string
	HealthAddress       string
	Namespace           string
	PomeriumConfigMap   string
	PomeriumNamespace   string
	PomeriumDeployments []string
	ServiceClass        string
}

var rootCmd = &cobra.Command{
	Use:   "pomerium-operator",
	Short: "pomerium-operator is a kubernetes operator for pomerium identity aware proxy",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := vcfg.Unmarshal(operatorCfg)
		if err != nil {
			return err
		}

		if operatorCfg.Debug {
			log.Debug()
		}

		logger.V(1).Info("started with config", "config", operatorCfg)

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
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

		deploymentManager := deploymentmanager.NewDeploymentManager(kClient, operatorCfg.PomeriumDeployments, operatorCfg.PomeriumNamespace)
		configManager.OnSave(deploymentManager.UpdateDeployments)

		if err := ingressController(o, configManager); err != nil {
			return err
		}
		if err := serviceController(o, configManager); err != nil {
			return err
		}

		if err := o.Add(configManager); err != nil {
			return err
		}

		if err := o.Start(); err != nil {
			logger.Error(err, "operator failed to start.  exiting")
			return err
		}

		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
func init() {
	rootCmd.PersistentFlags().Bool("debug", false, "Run in debug mode")
	rootCmd.PersistentFlags().StringP("namespace", "n", "", "Namespaces to monitor")
	rootCmd.PersistentFlags().String("pomerium-configmap", "pomerium", "Name of pomerium ConfigMap to maintain")
	rootCmd.PersistentFlags().String("pomerium-namespace", "kube-system", "Name of pomerium ConfigMap to maintain")
	rootCmd.PersistentFlags().String("base-config-file", "./pomerium-base.yaml", "Path to base configuration file")

	rootCmd.PersistentFlags().StringP("service-class", "s", "pomerium", "kubernetes.io/service.class to monitor")
	rootCmd.PersistentFlags().StringP("ingress-class", "i", "pomerium", "kubernetes.io/ingress.class to monitor")

	rootCmd.PersistentFlags().Bool("election", false, "Enable leader election (for running multiple controller replicas)")
	rootCmd.PersistentFlags().String("election-configmap", "operator-leader-pomerium", "Name of ConfigMap to use for leader election")
	rootCmd.PersistentFlags().String("election-namespace", "kube-system", "Namespace to use for leader election")
	rootCmd.PersistentFlags().String("metrics-address", "0", "Address for metrics listener.  Default disabled")
	rootCmd.PersistentFlags().String("health-address", "0", "Address for health check endpoint.  Default disabled")
	rootCmd.PersistentFlags().StringSlice("pomerium-deployments", []string{}, "List of Deployments in the pomerium-namespace to update when the [base-config-file] changes")

	err := bindViper(vcfg, rootCmd.PersistentFlags())
	if err != nil {
		fmt.Println(fmt.Errorf("failed to bind pflags: %w", err))
		os.Exit(1)
	}
}

func newRestClient(config *rest.Config) (client.Client, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid rest config passed")
	}
	// Set a timeout for any clients we create for object managers
	// TODO this should be configurable
	restConfig := rest.CopyConfig(config)
	restConfig.Timeout = 30 * time.Second

	c, err := client.New(restConfig, client.Options{})
	if err != nil {
		return nil, fmt.Errorf("failed to create client for config manager: %w", err)
	}

	return c, nil
}

func newConfigManager(kClient client.Client) (cm *configmanager.ConfigManager, err error) {
	baseConfigFile := operatorCfg.BaseConfigFile
	cm = configmanager.NewConfigManager(operatorCfg.PomeriumNamespace, operatorCfg.PomeriumConfigMap, kClient, time.Second*10)

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
	return controller.NewReconciler(ingressResource, operatorCfg.IngressClass, cm)
}

func serviceReconciler(cm *configmanager.ConfigManager) *controller.Reconciler {
	serviceResource := &corev1.Service{}
	return controller.NewReconciler(serviceResource, operatorCfg.ServiceClass, cm)
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
			KubeConfig:              kcfg,
			NameSpace:               operatorCfg.Namespace,
			ServiceClass:            operatorCfg.ServiceClass,
			IngressClass:            operatorCfg.IngressClass,
			MetricsBindAddress:      operatorCfg.MetricsAddress,
			HealthAddress:           operatorCfg.HealthAddress,
			LeaderElection:          operatorCfg.Election,
			LeaderElectionID:        operatorCfg.ElectionConfigMap,
			LeaderElectionNamespace: operatorCfg.ElectionNamespace,
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

	logger.V(1).Info("found kubeconfig", "api-server", kcfg.Host)
	return kcfg, nil
}

func bindViper(v *viper.Viper, flags *pflag.FlagSet) (err error) {
	flags.VisitAll(func(flag *pflag.Flag) {
		camelCasedFlag := strcase.ToCamel(flag.Name)
		snakeCasedFlag := strcase.ToScreamingSnake(flag.Name)

		err = v.BindPFlag(camelCasedFlag, flag)
		if err != nil {
			return
		}

		err = v.BindEnv(camelCasedFlag, snakeCasedFlag)
		if err != nil {
			return
		}
	})
	return nil
}
