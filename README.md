# argocd-applicationset-namespaces-generator-plugin

Namespaces Generator that discovers namespaces in a target cluster.

**THIS IS NOT FINISHED**

# Testing

```bash
go run ./... -v=0 --log-format=text server --local
curl -X POST -H "Content-Type: application/json" -d @testdata/request.json http://localhost:8080/api/v1/getparams.execute
```
