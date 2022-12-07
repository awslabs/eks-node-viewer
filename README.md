## Usage

### Install
```shell
go install github.com/awslabs/eks-node-viewer/cmd/eks-node-viewer@latest
```
### Standard
```shell
eks-node-viewer
```

### Karpenter Nodes Only
```shell
eks-node-viewer --nodeSelector "karpenter.sh/provisioner-name"
```

### Display CPU and Memory Usage
```shell
eks-node-viewer --resources cpu,memory
```
### Troubleshooting

#### NoCredentialProviders: no valid providers in chain. Deprecated.

This CLI relies on AWS credentials to access pricing data. You must have credentials configured via `~/aws/credentials`, `~/.aws/config`, environment variables, or some other credential provider chain.

See [credential provider documentation(https://docs.aws.amazon.com/sdk-for-go/api/aws/session/) for more.

## Development

### Building
```shell
$ make build
```
