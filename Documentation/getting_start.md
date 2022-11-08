# Getting start

## 🚀 Deploy the yurt-edgex-manager
This section tells you how to deploy the yurt-edgex-manger to a cluster

### 👟 Prepare a kubernetes cluster
To deploy or test the yurt-edgex-manager alone, we can start from a generic kubernetes cluster. i.e. you can create a cluster with 3 nodes by kind. More kind information can refer [kind usage](https://kind.sigs.k8s.io/)
Note: the kubernetes version must < v1.21.0
```
vim my-cluster-multi-node.yaml

kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
- role: worker
```
If you want to deploy other versions of the image, you can refer [image version](https://github.com/kubernetes-sigs/kind/releases/tag/v0.17.0).

```
kind create cluster --config my-cluster-multi-node.yaml --name openyurt --image kindest/node:v1.20.15@sha256:a32bf55309294120616886b5338f95dd98a2f7231519c7dedcec32ba29699394
```

### 👞 Deploy yurt-app-manager
The yurt-app-manager can be deployed by this command:

`kubectl apply -f https://raw.githubusercontent.com/openyurtio/yurt-app-manager/v0.6.0/config/setup/all_in_one.yaml`

or deployed with helm
```
git clone https://github.com/openyurtio/yurt-app-manager.git
cd yurt-app-manager
helm install yurt-app-manager ./charts/yurt-app-manager/
```
### 🏃 Deploy yurt-edgex-manager
Label the node for yurt-edgex-manager to delpoy
```
kubectl label node openyurt-worker openyurt.io/is-edge-worker="false"
```

Deploy the latest release of yurt-edgex-manager in OpenYurt cluster
`kubectl apply -f https://github.com/openyurtio/yurt-edgex-manager/releases/download/v0.2.0/yurt-edgex-manager.yaml`

or deployed with helm
```
git clone https://github.com/openyurtio/yurt-edgex-manager.git
cd yurt-edgex-manager
helm install yurt-edgex-manager ./charts/yurt-edgex-manager/
```

or deploy your own version by:
```
git clone https://github.com/openyurtio/yurt-edgex-manager
cd yurt-edgex-manager
make docker-build
kind load docker-image docker.io/openyurt/yurt-edgex-manager:latest --name openyurt 
make deploy
```

## 🛩️ Create Edgex
### ⛷️ Create nodepool for EdgeX deployment
```
cat <<EOF | kubectl apply -f -
apiVersion: apps.openyurt.io/v1alpha1
kind: NodePool
metadata:
  name: beijing
spec:
  type: Cloud
EOF

kubectl label node openyurt-worker apps.openyurt.io/desired-nodepool=beijing
```

### 🚢 Create Edgex
```
cat <<EOF | kubectl apply -f -
apiVersion: device.openyurt.io/v1alpha1
kind: EdgeX
metadata:
  name: edgex-sample-beijing
spec:
  version: jakarta
  poolName: beijing
EOF
```
Check the EdgeX status
```
kubectl get edgex
```

### ⏺️ Demo

![usage](usage.svg)

## 👩‍💻 Development
### Make binary and docker image
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

## 🧪 Test
After code change use can run the e2e test in the locally by
```
make test-e2e
```

For user who wants to debug could preserve the env by
```
SKIP_RESOURCE_CLEANUP=true make test-e2e
```