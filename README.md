# yurt-edgex-manager

Yurt-edgex-manager is openyurt to manager EdgeX lifecycle controller, it contains one CR (Custormer Resource) to reprents one EdgeX 
deployment. 
User now can install, upgrade, delete EdgeX in Openyurt cluster by just manipulating this CR. 

## Getting Start
### Deploy yurt-edgex-manager
Deploy latest yurt-edgex-manager in openyurt cluster

`kubectl apply -f https://raw.githubusercontent.com/openyurtio/yurt-edgex-manager/main/Documentation/yurt-edgex-manager.yaml`
### Usage

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
