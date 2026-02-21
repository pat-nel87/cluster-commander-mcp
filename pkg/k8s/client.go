package k8s

import (
	"fmt"
	"os"
	"path/filepath"

	apiextensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	metricsv "k8s.io/metrics/pkg/client/clientset/versioned"
)

// ClusterClient wraps Kubernetes client interfaces for cluster access.
type ClusterClient struct {
	Clientset           kubernetes.Interface
	MetricsClient       metricsv.Interface
	ApiextensionsClient apiextensionsclient.Interface
	Config              *rest.Config
	ContextName         string
}

// NewClusterClient creates a client from kubeconfig or in-cluster config.
// If contextName is empty, uses the current-context from kubeconfig.
func NewClusterClient(contextName string) (*ClusterClient, error) {
	// Try in-cluster first
	config, err := rest.InClusterConfig()
	if err != nil {
		// Fall back to kubeconfig
		kubeconfig := kubeconfigPath()

		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}
		overrides := &clientcmd.ConfigOverrides{}
		if contextName != "" {
			overrides.CurrentContext = contextName
		}

		clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)
		config, err = clientConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to build config: %w", err)
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create clientset: %w", err)
	}

	// Metrics API may not be available; don't fail hard
	metricsClient, _ := metricsv.NewForConfig(config)

	// API extensions client for CRDs; may not be available
	apiextClient, _ := apiextensionsclient.NewForConfig(config)

	return &ClusterClient{
		Clientset:           clientset,
		MetricsClient:       metricsClient,
		ApiextensionsClient: apiextClient,
		Config:              config,
		ContextName:         contextName,
	}, nil
}

// NewClusterClientForTesting creates a ClusterClient with injected fakes for unit tests.
func NewClusterClientForTesting(clientset kubernetes.Interface, metricsClient metricsv.Interface) *ClusterClient {
	return &ClusterClient{
		Clientset:     clientset,
		MetricsClient: metricsClient,
		ContextName:   "test-context",
	}
}

// NewClusterClientForTestingWithApiext creates a ClusterClient with injected fakes including apiextensions for unit tests.
func NewClusterClientForTestingWithApiext(clientset kubernetes.Interface, metricsClient metricsv.Interface, apiextClient apiextensionsclient.Interface) *ClusterClient {
	return &ClusterClient{
		Clientset:           clientset,
		MetricsClient:       metricsClient,
		ApiextensionsClient: apiextClient,
		ContextName:         "test-context",
	}
}

// ListAvailableContexts returns all contexts from the kubeconfig file
// and the name of the current context.
func ListAvailableContexts() ([]string, string, error) {
	kubeconfig := kubeconfigPath()

	config, err := clientcmd.LoadFromFile(kubeconfig)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	contexts := make([]string, 0, len(config.Contexts))
	for name := range config.Contexts {
		contexts = append(contexts, name)
	}
	return contexts, config.CurrentContext, nil
}

// kubeconfigPath returns the path to the kubeconfig file.
func kubeconfigPath() string {
	if p := os.Getenv("KUBECONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".kube", "config")
}
