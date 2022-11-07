

## Building
```sh
$ make build
```

## Running

### Standard
```shell
AWS_SDK_LOAD_CONFIG=true ./monitui
```


### Karpenter Nodes Only
```shell
AWS_SDK_LOAD_CONFIG=true ./monitui --nodeSelector "karpenter.sh/provisioner-name" 
```


### Display CPU and Memory Usage
```shell
AWS_SDK_LOAD_CONFIG=true  ./monitui  --resources cpu,memory
```