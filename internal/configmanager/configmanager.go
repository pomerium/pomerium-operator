package configmanager

import (
	"context"
	"fmt"
	"sync"
	"time"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/types"

	corev1 "k8s.io/api/core/v1"

	"github.com/pomerium/pomerium-operator/internal/log"

	pomeriumconfig "github.com/pomerium/pomerium/config"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var logger = log.L.WithValues("component", "configmanager")

const configKey = "config.yaml"

type ConfigManager struct {
	namespace    string
	configMap    string
	client       client.Client
	mutex        sync.RWMutex
	policyList   map[ResourceIdentifier][]pomeriumconfig.Policy
	baseConfig   []byte
	settleTicker *time.Ticker
	pendingSave  bool
}

func NewConfigManager(namespace string, configMap string, client client.Client, settlePeriod time.Duration) *ConfigManager {
	return &ConfigManager{
		namespace:    namespace,
		configMap:    configMap,
		client:       client,
		policyList:   make(map[ResourceIdentifier][]pomeriumconfig.Policy),
		settleTicker: time.NewTicker(settlePeriod),
	}
}

func (c *ConfigManager) Set(id ResourceIdentifier, policy []pomeriumconfig.Policy) {
	logger.V(1).Info("setting policy for resource", "id", id)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.policyList[id] = policy
	logger.Info("set policy for resource", "id", id)
	c.pendingSave = true
}

func (c *ConfigManager) Remove(id ResourceIdentifier) error {
	logger.V(1).Info("removing policy for resource", "id", id)

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if _, ok := c.policyList[id]; !ok {
		logger.V(1).Info("resource not fuond", "id", id)
		return fmt.Errorf("resource identifier '%s' not found", id)
	}

	delete(c.policyList, id)
	logger.Info("removed policy for resource", "id", id)
	c.pendingSave = true
	return nil
}

func (c *ConfigManager) Save() error {
	logger.V(1).Info("saving ConfigMap")

	var tmpOptions pomeriumconfig.Options

	tmpOptions, err := c.GetCurrentConfig()
	if err != nil {
		return fmt.Errorf("could not render current config: %w", err)
	}

	// Make sure we can load the target configmap
	configObj := &corev1.ConfigMap{}
	if err := c.client.Get(context.Background(), types.NamespacedName{Name: c.configMap, Namespace: c.namespace}, configObj); err != nil {
		err = fmt.Errorf("output configmap not found: %w", err)
		return err
	}

	configBytes, err := yaml.Marshal(tmpOptions)
	if err != nil {
		return fmt.Errorf("could not serialize config: %w", err)
	}

	configObj.Data = map[string]string{configKey: string(configBytes)}

	// TODO set deadline?
	// TODO use context from save?
	err = c.client.Update(context.Background(), configObj)
	if err != nil {
		return fmt.Errorf("failed to update configmap: %w", err)
	}

	logger.Info("successfully saved ConfigMap")
	c.pendingSave = false
	return nil
}

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

func (c *ConfigManager) GetPersistedConfig() (options pomeriumconfig.Options, err error) {
	configObj := &corev1.ConfigMap{}
	if err = c.client.Get(context.Background(), types.NamespacedName{Name: c.configMap, Namespace: c.namespace}, configObj); err != nil {
		return options, fmt.Errorf("output configmap not found: %w", err)
	}

	if err = yaml.Unmarshal([]byte(configObj.Data[configKey]), &options); err != nil {
		return options, fmt.Errorf("could not unmarshal config: %w", err)
	}

	return
}

func (c *ConfigManager) Start() {
	c.saveLoop()
}

func (c *ConfigManager) Stop() {
	c.settleTicker.Stop()
}

func (c *ConfigManager) saveLoop() {
	for {
		_, ok := <-c.settleTicker.C
		if c.pendingSave {
			err := c.Save()
			if err != nil {
				log.L.Error(err, "failed to save to configmap", "configmap", c.configMap)
			}
		}
		if !ok {
			break
		}
	}
}
