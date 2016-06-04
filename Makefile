all: image

code:
	glide install --strip-vendor --strip-vcs
	go install github.com/kopeio/aws-controller/cmd/aws-controller

test:
	go test -v github.com/kopeio/aws-controller/pkg/...

gofmt:
	gofmt -w -s cmd/
	gofmt -w -s pkg/

builder-image:
	docker build -f images/builder/Dockerfile -t builder .

build-in-docker: builder-image
	docker run -it -v `pwd`:/src builder /onbuild.sh

image: build-in-docker
	docker build -t kope/aws-controller  -f images/aws-controller/Dockerfile .

push: image
	docker push kope/aws-controller:latest
