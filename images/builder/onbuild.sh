#!/bin/bash

mkdir -p /go
export GOPATH=/go

mkdir -p /go/src/github.com/kopeio/
ln -s /src /go/src/github.com/kopeio/aws-controller

cd /go/src/github.com/kopeio/aws-controller
/usr/bin/glide install --strip-vendor --strip-vcs

go install github.com/kopeio/aws-controller/cmd/aws-controller

mkdir -p /src/.build/artifacts/
cp /go/bin/aws-controller /src/.build/artifacts/
