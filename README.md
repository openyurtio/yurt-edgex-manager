# yurt-edgex-manager

Yurt-edgex-manager is openyurt to manager EdgeX lifecycle controller, it contains one CR (Custormer Resource) to reprents one EdgeX
deployment.
User now can install, upgrade, delete EdgeX in Openyurt cluster by just manipulating this CR.

## Getting Start
### Make binary and docker-img
User can build the binary from source, golang is required.
```bash
# Only for go mod proxy
go env -w GOPROXY=https://goproxy.cn,direct
go mod download
make build
```
The generated binary will in the bin/manager

User can build the docker image. The docker is required and you can set IMG to the name you want

`IMG=openyurt/yurt-edgex-manger:v0.1 make docker-build`

### Deploy yurt-edgex-manager
Deploy latest yurt-edgex-manager in openyurt cluster

`kubectl apply -f https://github.com/openyurtio/yurt-edgex-manager/releases/download/v0.1.0/yurt-edgex-manager.yaml`
### Usage

![usage](./Documentation/usage.svg)

## Contributing

Contributions are welcome, whether by creating new issues or pull requests. See
our [contributing document](https://github.com/openyurtio/openyurt/blob/master/CONTRIBUTING.md) to get started.

## Contact

- Mailing List: openyurt@googlegroups.com
- Slack: [channel](https://join.slack.com/t/openyurt/shared_invite/zt-iw2lvjzm-MxLcBHWm01y1t2fiTD15Gw)
- Dingtalk Group (钉钉讨论群)

<div align="left">
    <img src="https://github.com/openyurtio/openyurt/blob/master/docs/img/ding.jpg" width=25% title="dingtalk">
</div>

## License
Yurt-edgex-manager is under the Apache 2.0 license. See the [LICENSE](LICENSE) file
for details. Certain implementations in Yurt-edgex-manager rely on the existing code
from [Kubernetes](https://github.com/kubernetes/kubernetes) and
[OpenKruise](https://github.com/openkruise/kruise) the credits go to the
original authors.
