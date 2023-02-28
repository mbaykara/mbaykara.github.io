---
template: overrides/main.html
title: Neuvector
---

# Neuvector

The neuvector is a security tool by SUSE.
Via [helm chart][1] in namespace `cattle-neuvector-system`
custom.yaml
```yaml
k3s:
  enabled: true
controller:
  ranchersso:
    enabled: true
global:
  cattle:
    url: https://rancher.com
```

[1]: https://artifacthub.io/packages/helm/neuvectorcharts/core
