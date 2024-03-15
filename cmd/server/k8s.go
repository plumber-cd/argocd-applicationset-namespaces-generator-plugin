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

func (c *ServerConfig) GetClient(req *RequestParameters) (*rest.Config, kubernetes.Interface, error) {
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
		url, err := url.Parse(*req.ClusterEndpoint)
		if err != nil {
			slog.Error("Failed to parse cluster endpoint", "endpoint", *req.ClusterEndpoint, "error", err)
		}
		config = &rest.Config{
			Host:            *req.ClusterEndpoint,
			BearerTokenFile: c.ServiceAccountTokenPath,
		}
		if req.UseLocalCA != nil && *req.UseLocalCA {
			tls := rest.TLSClientConfig{
				ServerName: url.Hostname(),
			}
			caData, err := base64.StdEncoding.DecodeString(c.ServiceAccountTlsCa)
			if err == nil {
				tls.CAData = caData
			} else {
				tls.CAFile = c.ServiceAccountTlsCa
			}
			config.TLSClientConfig = tls
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return config, nil, err
	}

	return config, clientset, nil
}
