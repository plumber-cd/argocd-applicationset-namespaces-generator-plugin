package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RequestParameters struct {
	ClusterEndpoint *string           `json:"clusterEndpoint,omitempty"`
	UseLocalCA      *bool             `json:"useLocalCA,omitempty"`
	LabelSelector   map[string]string `json:"labelSelector,omitempty"`
}

type RequestInput struct {
	Parameters *RequestParameters `json:"parameters,omitempty"`
}

type ResponseParameters struct {
	Namespace *string `json:"namespace,omitempty"`
}

type ResponseOutput struct {
	Parameters []*ResponseParameters `json:"parameters,omitempty"`
}

type ResponseBody struct {
	Output *ResponseOutput `json:"output,omitempty"`
}

func (c *ServerConfig) secretsHandler(ctx context.Context) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		slog.Debug("Received request", "address", r.RemoteAddr, "url", r.URL)
		if r.Method != http.MethodPost {
			slog.Debug("Method not allowed", "method", r.Method, "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusMethodNotAllowed)
			_, _ = w.Write([]byte("Method not allowed"))
			return
		}
		if r.Header.Get("Content-Type") != "application/json" {
			slog.Debug("Unsupported media type", "media-type", r.Header.Get("Content-Type"), "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusUnsupportedMediaType)
			_, _ = w.Write([]byte("Unsupported media type"))
			return
		}
		if c.ListenToken != "" && r.Header.Get("Authorization") != "Bearer "+c.ListenToken {
			slog.Debug("Unauthorized", "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte("Unauthorized"))
			return
		}

		input := RequestInput{}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			slog.Debug("Unable to read input json", "error", err, "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request"))
			return
		}

		if input.Parameters == nil {
			slog.Debug("No parameters provided", "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request"))
			return
		}

		_, k8s, err := c.GetClient(input.Parameters)
		if err != nil {
			slog.Error("Failed to get k8s client", "error", err, "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
			return
		}

		listOptions := metav1.ListOptions{}

		if input.Parameters != nil && input.Parameters.LabelSelector != nil {
			labels := []string{}
			for key, value := range input.Parameters.LabelSelector {
				labels = append(labels, key+"="+value)
			}
			listOptions.LabelSelector = strings.Join(labels, ",")
			slog.Debug("Using label selector", "labelSelector", listOptions.LabelSelector, "address", r.RemoteAddr, "url", r.URL)
		}

		namespaces, err := k8s.CoreV1().Namespaces().List(ctx, listOptions)
		if err != nil {
			slog.Error("Failed to list namespaces", "error", err, "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
			return
		}

		output := ResponseBody{
			Output: &ResponseOutput{
				Parameters: []*ResponseParameters{},
			},
		}

		for _, ns := range namespaces.Items {
			output.Output.Parameters = append(output.Output.Parameters, &ResponseParameters{
				Namespace: &ns.Name,
			})
		}

		slog.Debug("Returning response", "address", r.RemoteAddr, "url", r.URL, "output", output)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(output); err != nil {
			slog.Error("Failed to encode response", "error", err, "address", r.RemoteAddr, "url", r.URL)
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		}
	}
}
