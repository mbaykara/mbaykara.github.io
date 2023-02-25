---
template: overrides/main.html
title: Air-Gapped
---
# Air-Gapped Installation

On control plane

`/etc/rancher/rke2/config.yaml` 
```yaml
tls-san:
  - rancher-spim.sva.wtf
  - 167.235.107.218
```

On Workers

`/etc/rancher/rke2/config.yaml` 
```yaml

server: https://rancher-spim.sva.wtf:9345
token: <abgefragter Token>
tls-san:
  - rancher-spim.sva.wtf
  - 167.235.107.218
```