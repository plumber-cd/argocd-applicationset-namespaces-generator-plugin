package server

import (
	"encoding/base64"
	"log/slog"
	"net/url"

	"errors"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func (c *ServerConfig) GetClient(req *PluginParameters) (*rest.Config, kubernetes.Interface, error) {
	var config *rest.Config
	var err error

	if c.Local {
		slog.Debug("We are in --local mode")
		kubeconfigPath := ""
		if os.Getenv("KUBECONFIG") != "" {
			slog.Debug("Found KUBECONFIG environment variable", "KUBECONFIG", os.Getenv("KUBECONFIG"))
			kubeconfigPath = os.Getenv("KUBECONFIG")
		} else if home := homedir.HomeDir(); home != "" {
			slog.Debug("Falling back to user home", "HOME", home)
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}

		if kubeconfigPath == "" {
			return nil, nil, errors.New("Cannot find KUBECONFIG or default kubeconfig file")
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			return nil, nil, err
		}
	} else {
		var serviceAccountTokenPath string
		if tokenPath, ok := c.ServiceAccountTokenPathsAsMap[*req.ClusterName]; ok {
			slog.Debug("Found token path for cluster", "cluster", req.ClusterName, "token-path", tokenPath)
			serviceAccountTokenPath = tokenPath
		} else {
			slog.Debug("Using default token path", "cluster", req.ClusterName, "token-path", c.ServiceAccountTokenPathsAsMap["*"])
			serviceAccountTokenPath = c.ServiceAccountTokenPathsAsMap["*"]
		}

		url, err := url.Parse(*req.ClusterEndpoint)
		if err != nil {
			slog.Error("Failed to parse cluster endpoint", "cluster", req.ClusterName, "endpoint", req.ClusterEndpoint, "error", err)
			return nil, nil, err
		}
		config = &rest.Config{
			Host:            *req.ClusterEndpoint,
			BearerTokenFile: serviceAccountTokenPath,
		}
		tls := rest.TLSClientConfig{
			ServerName: url.Hostname(),
		}
		if req.ClusterCA != nil && *req.ClusterCA != "" {
			slog.Debug("Using cluster CA from the request", "cluster", req.ClusterName, "clusterEndpoint", req.ClusterEndpoint)
			ca := *req.ClusterCA
			caData, err := base64.StdEncoding.DecodeString(ca)
			if err != nil {
				slog.Error("Failed to decode cluster CA from the request", "error", err)
				return nil, nil, err
			}
			tls.CAData = caData
		} else {
			slog.Debug("Using cluster CA from the config", "cluster", req.ClusterName, "clusterEndpoint", req.ClusterEndpoint)
			ca := c.ServiceAccountTlsCa
			caData, err := base64.StdEncoding.DecodeString(ca)
			if err == nil {
				tls.CAData = caData
			} else {
				tls.CAFile = c.ServiceAccountTlsCa
			}
		}
		config.TLSClientConfig = tls
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return config, nil, err
	}

	return config, clientset, nil
}
