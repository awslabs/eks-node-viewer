

## Building
```sh
$ make build
```

## Running

We read from the shared credentials file by default (`~/.aws/credentials`).  To also read from your `~/.aws/config`, set `AWS_SDK_LOAD_CONFIG=true`.

### Standard
```shell
./eks-node-viewer
```


### Karpenter Nodes Only
```shell
./eks-node-viewer --nodeSelector "karpenter.sh/provisioner-name" 
```


### Display CPU and Memory Usage
```shell
./eks-node-viewer  --resources cpu,memory
```