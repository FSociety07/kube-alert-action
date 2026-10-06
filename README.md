# TODO

* querier component
* leader election
* rate-limit google chat calls
* parallel action execution

# INSTALL

docker pull sawn37/kube-alert-action:latest

https://hub.docker.com/repository/docker/sawn37/kube-alert-action

## From K8S Manifests

kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/config/crd/sawnt.xyz_actionmaps.yaml
kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/config/crd/sawnt.xyz_alertevents.yaml
kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/kube-manifests.yaml


## Helm
helm chart versions: 
https://github.com/fsociety07/kube-alert-action/pkgs/container/kube-alert-action

helm install kube-alert-action oci://ghcr.io/fsociety07/kube-alert-action -n <namepsace> --create-namespace


# Alert payload format
```
[
	{"targetNamespace":"prod","container":"payment-api","pod":"payment-api-f35ds-242f","metric":"heap-memory-usage-bytes"}, 
	{...}
]
```

# Actionmap CR Manifest

```yaml
apiVersion: sawnt.xyz/v1alpha1
kind: ActionMap
metadata:
  name: prod-container-actions
  namespace: prod
spec:
  rules:
    - metric: heap-memory-usage-bytes
      action: /scripts/heapdump.sh
      executeFrom: targetPod
    - metric: cpu-usage-seconds
      action: /scripts/threaddump.sh
      executeFrom: targetPod
    - metric: corrupted-jcs
      action: /scripts/graceful-restart.sh
      executeFrom: self
```
