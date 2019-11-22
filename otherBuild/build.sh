#!/bin/bash

VERSION=0.0

echo "This is a poor way of doing it, but it works"

cd ..

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o otherbuild/manager main.go

cd -

docker build -t thebsdbox/capp:$VERSION .
docker push thebsdbox/capp:$VERSION