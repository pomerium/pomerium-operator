package configmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pomerium/pomerium-operator/internal/log"
	pomeriumconfig "github.com/pomerium/pomerium/config"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var logger = log.L.WithValues("component", "configmanager")

const configKey = "config.yaml"

// ConfigManager tracks policy groups related to a given ResourceIdentifier and handles update to a Pomerium config Secret via the API server
//
// ConfigManager accepts a baseConfig which will be merged into the persisted configuration
//
// Configuration can be persisted at intervals or on-demand.  Set() and Remove() operations are stored in memory only until a Save() or Start() loop
// persist the configuration.
type ConfigManager struct {
	namespace    string
	secret       string
	client       client.Client
	mutex        sync.RWMutex
	policyList   map[ResourceIdentifier][]pomeriumconfig.Policy
	baseConfig   []byte
	settleTicker *time.Ticker
	onSaves      []ConfigReceiver
}

// NewConfigManager returns a ConfigManager which uses client to update secret in namespace at settlePeriod interval if
// running the save loop via Start()
func NewConfigManager(namespace string, secret string, client client.Client, settlePeriod time.Duration) *ConfigManager {
	return &ConfigManager{
		namespace:    namespace,
		secret:       secret,
		client:       client,
		policyList:   make(map[ResourceIdentifier][]pomeriumconfig.Policy),
		settleTicker: time.NewTicker(settlePeriod),
	}
}

// Set Adds or replaces the list of policies associated with a given ResourceIdentifier id
func (c *ConfigManager) Set(id ResourceIdentifier, policy []pomeriumconfig.Policy) {
	logger.V(1).Info("setting policy for resource", "id", id)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.policyList[id] = policy
	logger.Info("set policy for resource", "id", id)
}

// Remove Deletes the list of policies associated with a given ResourceIdentifier id
func (c *ConfigManager) Remove(id ResourceIdentifier) error {
	logger.V(1).Info("removing policy for resource", "id", id)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.policyList[id]; !ok {
		logger.V(1).Info("resource not found", "id", id)
		return nil
	}

	delete(c.policyList, id)
	logger.Info("removed policy for resource", "id", id)
	return nil
}

// Save immediately flushes the current configuration to the API server
func (c *ConfigManager) Save() error {
	logger.V(1).Info("updating config Secret")

	var tmpOptions pomeriumconfig.Options

	tmpOptions, err := c.GetCurrentConfig()
	if err != nil {
		return fmt.Errorf("could not render current config: %w", err)
	}

	secretObj := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: c.secret, Namespace: c.namespace}}
	op, err := controllerutil.CreateOrUpdate(context.TODO(), c.client, secretObj, func() error {
		configBytes, err := yaml.Marshal(tmpOptions)
		if err != nil {
			return fmt.Errorf("could not serialize config: %w", err)
		}

		secretObj.Data = map[string][]byte{configKey: configBytes}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update secret: %w", err)
	}

	logger.WithValues("operation", op).V(1).Info("update config Secret result")

	if op == controllerutil.OperationResultUpdated || op == controllerutil.OperationResultCreated {
		logger.Info("successfully saved Secret")
		c.callOnSaves(tmpOptions)
	}
	return nil
}

// SetBaseConfig Allows arbitrary Pomerium configuration to be set with the resource based policies being saved.  This allows the user to
// still set all Pomerium options in a config file, even though it is being managed by ConfigManager.
func (c *ConfigManager) SetBaseConfig(configBytes []byte) error {
	err := yaml.Unmarshal(configBytes, &pomeriumconfig.Options{})
	if err != nil {
		return fmt.Errorf("failed to unmarshal configuration: %w", err)
	}
	c.baseConfig = configBytes
	return nil
}

func (c *ConfigManager) getBaseConfig() (options pomeriumconfig.Options, err error) {
	err = yaml.Unmarshal(c.baseConfig, &options)
	if err != nil {
		return options, fmt.Errorf("failed to load base config: %w", err)
	}
	return
}

// GetCurrentConfig retrieves the current in-memory configuration from ConfigManager
func (c *ConfigManager) GetCurrentConfig() (options pomeriumconfig.Options, err error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Create an Options struct from base config
	options, err = c.getBaseConfig()
	if err != nil {
		logger.Error(err, "could not load base configuration")
		return options, fmt.Errorf("could not load base configuration: %w", err)
	}

	// Attach policies
	for _, policy := range c.policyList {
		options.Policies = append(options.Policies, policy...)
	}

	return
}

// GetPersistedConfig retrieves the currently persisted config from the API server
func (c *ConfigManager) GetPersistedConfig() (options pomeriumconfig.Options, err error) {
	secretObj := &corev1.Secret{}
	if err = c.client.Get(context.Background(), types.NamespacedName{Name: c.secret, Namespace: c.namespace}, secretObj); err != nil {
		return options, fmt.Errorf("output secret not found: %w", err)
	}

	if err = yaml.Unmarshal([]byte(secretObj.Data[configKey]), &options); err != nil {
		return options, fmt.Errorf("could not unmarshal config: %w", err)
	}

	return
}

// Start implements manager.Runnable
//
//begins the periodic save loop to persist in-memory configuration to the API
func (c *ConfigManager) Start(stopCh <-chan struct{}) error {
	for {
		select {
		case <-stopCh:
			c.loopSave()
			return nil
		case <-c.settleTicker.C:
			c.loopSave()
		}
	}
}

func (c *ConfigManager) loopSave() {
	err := c.Save()
	if err != nil {
		log.L.Error(err, "failed to save to secret", "secret", c.secret)
	}
}

// NeedLeaderElection implements manager.LeaderElectionRunnable.
//
// When ConfigManager is added to a controller-manager, this delays
// running Start() until leadership is established
func (c *ConfigManager) NeedLeaderElection() bool {
	return true
}

// ConfigReceiver is called with the stored configuration of the ConfigurationManager
type ConfigReceiver func(pomeriumconfig.Options)

// OnSave adds a ConfigReceiver function to call when ConfigManager has successfully committed
// configuration to storage.
func (c *ConfigManager) OnSave(f ConfigReceiver) {
	logger.V(1).Info("calling OnSave hooks")
	c.onSaves = append(c.onSaves, f)
}

func (c *ConfigManager) callOnSaves(config pomeriumconfig.Options) {
	for _, f := range c.onSaves {
		f(config)
	}
}
