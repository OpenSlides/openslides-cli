package client

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/OpenSlides/openslides-cli/internal/logger"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Client struct {
	clientset *kubernetes.Clientset
	config    *rest.Config
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
