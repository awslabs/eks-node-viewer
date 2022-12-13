[![GitHub License](https://img.shields.io/badge/License-Apache%202.0-ff69b4.svg)](https://github.com/awslabs/eks-node-viewer/blob/main/LICENSE)
[![contributions welcome](https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat)](https://github.com/aws/karpenter/issues)

## Usage

`eks-node-viewer` is a tool for visualizing dynamic node usage within a cluster.  It was originally developed as an internal tool at AWS for demonstrating consolidation with [Karpenter](https://karpenter.sh/).  

![](./.static/screenshot.png)

### Talks Using eks-node-viewer

- [Containers from the Couch: Workload Consolidation with Karpenter](https://www.youtube.com/watch?v=BnksdJ3oOEs)
- [AWS re:Invent 2022 - Kubernetes virtually anywhere, for everyone](https://www.youtube.com/watch?v=OB7IZolZk78)

### Installation

```shell
go install github.com/awslabs/eks-node-viewer/cmd/eks-node-viewer@latest
```

Note: This will install it to your `GOBIN` directory, typically `~/go/bin` if it is unconfigured.

## Usage
```shell
Usage of ./eks-node-viewer:
  -disable-pricing
    	Disable pricing lookups
  -kubeconfig string
    	(optional) absolute path to the kubeconfig file (default "~/.kube/config")
  -nodeSelector string
    	Node label selector used to filter nodes, if empty all nodes are selected
  -resources string
    	List of comma separated resources to monitor (default "cpu")
```

### Examples
```shell
# Standard usage
eks-node-viewer
# Karenter nodes only
eks-node-viewer --nodeSelector "karpenter.sh/provisioner-name"
# Display both CPU and Memory Usage 
eks-node-viewer --resources cpu,memory
# Specify a particular AWS profile and region 
AWS_PROFILE=myprofile AWS_REGION=us-west-2 
```
### Troubleshooting

#### NoCredentialProviders: no valid providers in chain. Deprecated.

This CLI relies on AWS credentials to access pricing data if you don't use the `--disable-pricing` option. You must have credentials configured via `~/aws/credentials`, `~/.aws/config`, environment variables, or some other credential provider chain.

See [credential provider documentation](https://docs.aws.amazon.com/sdk-for-go/api/aws/session/) for more.

#### I get an error of `creating client, exec plugin: invalid apiVersion "client.authentication.k8s.io/v1alpha1"`

Updating your AWS cli to the latest version and [updating your kubeconfig](https://docs.aws.amazon.com/cli/latest/reference/eks/update-kubeconfig.html) should resolve this issue.   

## Development

### Building
```shell
$ make build
```
