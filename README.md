

## Building
```sh
$ make build
```

## Running

### Standard
```shell
AWS_SDK_LOAD_CONFIG=true ./eks-node-viewer
```


### Karpenter Nodes Only
```shell
AWS_SDK_LOAD_CONFIG=true ./eks-node-viewer --nodeSelector "karpenter.sh/provisioner-name" 
```


### Display CPU and Memory Usage
```shell
AWS_SDK_LOAD_CONFIG=true  ./eks-node-viewer  --resources cpu,memory
```