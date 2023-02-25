---
template: overrides/main.html
title: ArgoCD
---
# ArgoCD
## Helm installation:


## Multi-Tenancy
Install the ArgoCD in a separate cluster rather than adding on the application's clusters. The install-ha.sh should be applied for High Availability version. The HA mod require at leas 3 node for redis replicas respectively.

Add each
`secret.yaml`


```yml
apiVersion: v1
kind: Secret
metadata:
  name: mycluster-secret
  labels:
    argocd.argoproj.io/secret-type: cluster
type: Opaque
stringData:
  name: mycluster.com
  server: https://mycluster.com
  config: |
    {
      "bearerToken": "<authentication token>",
      "tlsClientConfig": {
        "insecure": false,
        "caData": "<base64 encoded certificate>"
      }
    }
```

* For multitanent cluster to add new cluster to argo, UI is not working so you have to add it via SECRET object.


