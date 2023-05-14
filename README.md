# runtime-copilot
The main function of the runtime copilot is to assist the operation of the container runtime component (containerd), specifically for adding or deleting non-safe registries.

## Usage

[Helm](https://helm.sh) must be installed to use the charts.  Please refer to
Helm's [documentation](https://helm.sh/docs) to get started.

Once Helm has been set up correctly, add the repo as follows:

helm repo add runtime-copilot https://lengrongfu.github.io/runtime-copilot

If you had already added this repo earlier, run `helm repo update` to retrieve
the latest versions of the packages.  You can then run `helm search repo
runtime-copilot` to see the charts.

To install the <chart-name> chart:

    helm install runtime-copilot runtime-copilot/runtime-copilot

To uninstall the chart:

    helm delete runtime-copilot