# argocd-applicationset-namespaces-generator-plugin

Namespaces Generator that discovers namespaces in a target cluster.

It can be used as ArgoCD ApplicationSet plugin https://argo-cd.readthedocs.io/en/stable/operator-manual/applicationset/Generators-Plugin/.

It can discover existing namespaces in the cluster to produce an app per each namespace.

## Assumptions and prerequisites

- You are using JWT authentication to your clusters (i.e. Downward API tokens mounted to pods)
- If using external clusters, you must populate cluster annotation with its Certificate Authority

## Usage

Deploy using example from `testdata/manifest.yaml`.

Here's an example to use together with clusters generator via matrix generator:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: test-namespaces-generator
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
  - matrix:
      generators:
      - clusters: {}
      - plugin:
          configMapRef:
            name: argocd-applicationset-namespaces-generator-plugin
          input:
            parameters:
              clusterName: "{{ .name }}"
              clusterEndpoint: "{{ .server }}"
              # Use annotation with CA data in base64 format from the cluster
              clusterCA: '{{ index .metadata.annotations "my-org.com/cluster-ca" }}'
              # Optional, if not set means all namespaces
              labelSelector:
                some-label: some-value
  template:
    metadata:
      name: '{{ .name }}-{{ .namespace }}-test-namespaces-generator'
      namespace: '{{ .namespace }}'
    spec:
      source:
        repoURL: https://github.com/plumber-cd/argocd-applicationset-namespaces-generator-plugin
        targetRevision: main
        path: testdata
        kustomize:
          namespace: '{{ .namespace }}'
      destination:
        server: '{{ .server }}'
        namespace: '{{ .namespace }}'
      syncPolicy:
        syncOptions:
        # On mass propagation it is probably a good idea to make sure not to accidentally override resources
        - FailOnSharedResource=true
```

# Testing

```bash
go run ./... -v=0 --log-format=text server --local
curl -X POST -H "Content-Type: application/json" -d @testdata/request.json http://localhost:8080/api/v1/getparams.execute
```
