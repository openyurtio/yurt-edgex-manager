# Getting start

## 🚀 Deploy the yurt-edgex-manager
This section tells you how to deploy the yurt-edgex-manger to a cluster

### 👟 Prepare a kubernetes cluster
To deploy or test the yurt-edgex-manager alone, we can start from a generic kubernetes cluster. i.e. you can create a cluster with 3 nodes by kind. More kind information can refer [kind usage](https://kind.sigs.k8s.io/)
Note: the kubernetes version must < v1.21.0
```
kind create cluster
```

### 👞 Deploy yurt-app-manager
The yurt-app-manager can be deployed by this command:

`kubectl apply -f https://raw.githubusercontent.com/openyurtio/yurt-app-manager/v0.5.0/config/setup/all_in_one.yaml`

### 🏃 Deploy yurt-edgex-manager
Label the node for yurt-edgex-manager to delpoy
```
kubectl label node openyurt-worker openyurt.io/is-edge-worker="false"
```

Deploy the latest release of yurt-edgex-manager in OpenYurt cluster
`kubectl apply -f https://github.com/openyurtio/yurt-edgex-manager/releases/download/v0.2.0/yurt-edgex-manager.yaml`

or deploy your own version by:
```
git clone https://github.com/openyurtio/yurt-edgex-manager
cd yurt-edgex-manager
IMG=user/yurt-edgex-manager:dev make docker-push-mutiarch
IMG=user/yurt-edgex-manager:dev make deploy
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
  poolname: beijing
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