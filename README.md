![Github Ci Action](https://github.com/copilot-io/runtime-copilot/actions/workflows/ci.yml/badge.svg)![Github Image Action](https://github.com/copilot-io/runtime-copilot/actions/workflows/push-images.yml/badge.svg)![Github Chart Action](https://github.com/copilot-io/runtime-copilot/actions/workflows/release.yml/badge.svg)

# runtime-copilot
The main function of the runtime copilot is to assist the operation of the container runtime component (containerd), specifically for adding or deleting non-safe registries.

## Introduction

This project is a runtime copilot, auxiliary manager runtime, current function introduce the config insecure registry to runtime(such as: containerd、docker、cri-o), It mainly has the following functions:



- [ ] Manager insecure registry in runtime.
- [ ] Upgrade runtime version.
- [ ] Replace runtime with another runtime.
- [ ] Manager runtime plugins.

### Manager insecure registry

| Runtime | Support |
| --- | --- |
| containerd | Yes |
| docker | No |
| cri-o | No |

## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

    helm repo add runtime-copilot https://copilot-io.github.io/runtime-copilot

If you is first time to use this repo, you need to run command as follows:

    helm repo update
    helm search repo runtime-copilot

To install the runtime-copilot chart:

    helm install runtime-copilot runtime-copilot/runtime-copilot --namespace runtime-copilot

To uninstall the chart:

    helm delete runtime-copilot --namespace runtime-copilot

## Examples

We add `10.6..112.191` this insecret registry to containerd, we can define yaml content follow file.

```yaml
apiVersion: registry.runtime.x-copilot.io/v1alpha1
kind: RegistryConfigs
metadata:
  name: registryconfigs-sample
spec:
  selector:
    matchLabels:
      app: registryconfigs-sample
  template:
    spec:
      hostConfigs:
        - server: "https://10.6.112.191"
          capabilities:
            - pull
            - push
            - resolve
          skip_verify: true
```


After executing `kubectl apply`, the following `hosts.toml` file will be generated on each node, the content is as follows: 

```yaml
server = "https://10.6.112.191"

[host]
  [host."https://10.6.112.191"]
    capabilities = ["pull", "push", "resolve"]
    skip_verify = true
    override_path = false
```

