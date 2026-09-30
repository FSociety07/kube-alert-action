TODO

helm packaging

leader election

INSTALL

docker pull sawn37/kube-alert-action:latest

kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/config/crd/sawnt.xyz_actionmaps.yaml
kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/config/crd/sawnt.xyz_alertevents.yaml
kubectl apply -f https://raw.githubusercontent.com/FSociety07/kube-alert-action/refs/heads/main/kube-manifests.yaml


helm chart versions
https://github.com/fsociety07/kube-alert-action/pkgs/container/kube-alert-action

helm install kube-alert-action \
  oci://ghcr.io/fsociety07/kube-alert-action \
  --version 0.1.0
