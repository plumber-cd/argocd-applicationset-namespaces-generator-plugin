package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ServerConfig struct {
	ListenAddress string `mapstructure:"listen-address"`
	ListenToken   string `mapstructure:"listen-token"`
	ListenTlsCa   string `mapstructure:"listen-tls-ca"`
	ListenTlsCrt  string `mapstructure:"listen-tls-crt"`
	ListenTlsKey  string `mapstructure:"listen-tls-key"`

	Local bool `mapstructure:"local"`

	ServiceAccountTlsCa           string   `mapstructure:"service-account-tls-ca"`
	ServiceAccountTokenPaths      []string `mapstructure:"service-account-token-paths"`
	ServiceAccountTokenPathsAsMap map[string]string
}

func init() {
	Cmd.PersistentFlags().String("listen-address", ":8080", "Local address to listen on")
	Cmd.PersistentFlags().String("listen-token", "", "Bearer token to authenticate requests (if needed)")
	Cmd.PersistentFlags().String("listen-tls-ca", "", "TLS CA for server (if needed)")
	Cmd.PersistentFlags().String("listen-tls-crt", "", "TLS cert for the server (if needed)")
	Cmd.PersistentFlags().String("listen-tls-key", "", "TLS key for the server (if needed)")

	Cmd.PersistentFlags().Bool("local", false, "Enable to use local kubectl context (for debugging)")

	Cmd.PersistentFlags().String("service-account-tls-ca", "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt", "Path or base64 to ca.crt for cluster endpoint (if needed, ignored in --local mode)")
	Cmd.PersistentFlags().StringArray("service-account-token-paths", []string{"*=/var/run/secrets/kubernetes.io/serviceaccount/token"}, "Paths to a token file (ignored in --local mode)")
}

var Cmd = &cobra.Command{
	Use:   "server",
	Short: "Start plugin server",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		config := ServerConfig{}
		if err := viper.Unmarshal(&config); err != nil {
			return err
		}

		slog.Debug("Received list of token paths", "token-paths", config.ServiceAccountTokenPaths)
		config.ServiceAccountTokenPathsAsMap = make(map[string]string)
		_ServiceAccountTokenPath := []string{}
		for _, v := range config.ServiceAccountTokenPaths {
			parts := strings.Split(v, ",")
			_ServiceAccountTokenPath = append(_ServiceAccountTokenPath, parts...)
		}
		for _, v := range _ServiceAccountTokenPath {
			parts := strings.SplitN(strings.TrimSpace(v), "=", 2)
			if len(parts) != 2 {
				return errors.New("Invalid service-account-token-path format")
			}
			config.ServiceAccountTokenPathsAsMap[parts[0]] = parts[1]
		}
		slog.Debug("Resulting token paths as map", "token-paths", config.ServiceAccountTokenPathsAsMap)

		http.HandleFunc("/api/v1/getparams.execute", config.secretsHandler(ctx))

		if config.ListenTlsCrt != "" || config.ListenTlsKey != "" {
			cert, err := tls.LoadX509KeyPair(config.ListenTlsCrt, config.ListenTlsKey)
			if err != nil {
				slog.Error("server: load cert", "error", err)
			}

			tlsConfig := &tls.Config{
				Certificates: []tls.Certificate{cert},
				ClientAuth:   tls.RequireAndVerifyClientCert,
			}

			if config.ListenTlsCa != "" {
				caCert, err := os.ReadFile("ca.crt")
				if err != nil {
					slog.Error("server: read ca cert", "error", err)
				}
				caCertPool := x509.NewCertPool()
				caCertPool.AppendCertsFromPEM(caCert)
				tlsConfig.ClientCAs = caCertPool
			}

			server := &http.Server{
				Addr:      config.ListenAddress,
				TLSConfig: tlsConfig,
			}

			slog.Info("Server starting with TLS...", "listenAddress", config.ListenAddress)
			log.Fatal(server.ListenAndServeTLS("", ""))
		} else {
			slog.Info("Server starting...", "listenAddress", config.ListenAddress)
			if err := http.ListenAndServe(config.ListenAddress, nil); err != nil {
				slog.Error("Server Failure", "err", err)
				return err
			}
		}

		return nil
	},
}
