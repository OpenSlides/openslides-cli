package client

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config

	dynamicClient dynamic.Interface
	dynamicOnce   sync.Once
	dynamicErr    error

	restMapper meta.RESTMapper
	mapperOnce sync.Once
	mapperErr  error
}

// New creates a Kubernetes client
func New(kubeconfigPath string) (*Client, error) {
	var config *rest.Config
	var err error
	var source string

	if kubeconfigPath != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load kubeconfig from %s: %w", kubeconfigPath, err)
		}
		source = fmt.Sprintf("kubeconfig: %s", kubeconfigPath)
	} else {
		config, err = rest.InClusterConfig()
		if err == nil {
			source = "in-cluster service account"
		} else {
			home := os.Getenv("HOME")
			if home == "" {
				return nil, fmt.Errorf("failed to get in-cluster config and HOME env var not set")
			}
			kubeconfigPath = filepath.Join(home, ".kube", "config")
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create k8s config: not running in-cluster and no valid kubeconfig found at %s: %w", kubeconfigPath, err)
			}
			source = fmt.Sprintf("kubeconfig: %s", kubeconfigPath)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	logger.Debug("Kubernetes client initialized from %s", source)

	return &Client{
		clientset: clientset,
		config:    config,
	}, nil
}

// Clientset returns the underlying Kubernetes clientset
func (c *Client) Clientset() *kubernetes.Clientset {
	return c.clientset
}

// Config returns the underlying Kubernetes config
func (c *Client) Config() *rest.Config {
	return c.config
}

// Dynamic returns a cached dynamic client, creating it on first call
func (c *Client) Dynamic() (dynamic.Interface, error) {
	c.dynamicOnce.Do(func() {
		c.dynamicClient, c.dynamicErr = dynamic.NewForConfig(c.config)
		if c.dynamicErr == nil {
			logger.Debug("Dynamic client initialized")
		}
	})
	return c.dynamicClient, c.dynamicErr
}

// RESTMapper returns a cached REST mapper, creating it on first call
func (c *Client) RESTMapper() (meta.RESTMapper, error) {
	c.mapperOnce.Do(func() {
		apiGroupResources, err := restmapper.GetAPIGroupResources(c.clientset.Discovery())
		if err != nil {
			c.mapperErr = fmt.Errorf("getting API group resources: %w", err)
			return
		}

		c.restMapper = restmapper.NewDiscoveryRESTMapper(apiGroupResources)

		logger.Debug("REST mapper initialized")
	})

	return c.restMapper, c.mapperErr
}
