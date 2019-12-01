#!/bin/bash

VERSION=v0.1.1

echo "This is a terrible way for building the controller, but feel free to come up with something nicer"

cd ../..

echo "Generating CRDs"

make generate

cat config/crd/bases/* > cluster-api-plunder-components.yaml

cat <<EOF >> cluster-api-plunder-components.yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    control-plane: capp-controller-manager
  name: capp-controller-manager
  namespace: capi-system
spec:
  replicas: 1
  selector:
    matchLabels:
      control-plane: capp-controller-manager
  template:
    metadata:
      labels:
        control-plane: capp-controller-manager
    spec:
      containers:
      - args:
        - --metrics-addr=127.0.0.1:8080
        - --enable-leader-election
        image: thebsdbox/capp:$VERSION
        name: manager
        volumeMounts:
        - name: plunderyaml
          mountPath: "/plunderclient.yaml"
          subPath: "plunderclient.yaml"
      terminationGracePeriodSeconds: 10
      volumes:
      - name: plunderyaml
        secret:
          secretName: plunder
EOF

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o hack/dockerBuild/manager main.go

cd -

docker build -t thebsdbox/capp:$VERSION .
docker push thebsdbox/capp:$VERSION

echo "Tidying up working directory, then inspect the docker images"
rm manager
docker images | grep capp
