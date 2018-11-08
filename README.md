# gRPC
Project to play with proto defintions

## Reference
[Practical gRPC Book](https://www.oreilly.com/library/view/practical-grpc/9781939902580/)
[Samples from book](https://github.com/backstopmedia/gRPC-book-example)

## Plugin installation
All gRPC plugins (other than for Go, Java, and Dart) are in the main
gRPC repo, which is also where you will find installation
instructions: https://github.com/grpc/grpc.

Getting a working environment difficult, for example here are the
steps to get
[grpc-gateway installed](https://github.com/grpc-ecosystem/grpc-gateway#installation).
You can look at docker images that will give you a working environment:
https://github.com/namely/docker-protoc

There is also one bundled with InfobloxOpen that includes all you need
for Atlas Applications,
[Atlas Docker](infoblox/atlas-gentool:latest).
Here is the pattern for using it in a Makefile:
```sh
# configuration for the protobuf gentool
SRCROOT_ON_HOST      := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
SRCROOT_IN_CONTAINER := /go/src/$(PROJECT_ROOT)
DOCKER_RUNNER        := docker run --rm
DOCKER_RUNNER        += -v $(SRCROOT_ON_HOST):$(SRCROOT_IN_CONTAINER)
DOCKER_GENERATOR     := infoblox/atlas-gentool:latest
GENERATOR            := $(DOCKER_RUNNER) $(DOCKER_GENERATOR)

.PHONY: protobuf
protobuf:
	@$(GENERATOR) \
	--go_out=plugins=grpc:. \
	--grpc-gateway_out=logtostderr=true:. \
	--gorm_out="engine=postgres:." \
	--swagger_out="atlas_patch=true:." \
	--atlas-query-validate_out=. \
	--atlas-validate_out="." \
	--validate_out="lang=go:." 	$(PROJECT_ROOT)/proto/sfapi.proto
```
## sample1 proto
In order to generate code for services, we must use a protoc plugin.
The gRPC documentation site has tutorials for each supported
language (https://grpc.io/docs/tutorials/). By reviewing the tutorials
for the language you are using, you will find how to use the gRPC plugin
to generate gRPC-specific code for services.
```sh
go get github.com/golang/protobuf/protoc-gen-go
```

Generating Go code requires a plugin: protoc-gen-go. This plugin provides
standard output, for messages and enums, and also provides gRPC output if
you specify plugins=grpc in the --go_out argument when invoking protoc.
```sh
protoc --go_out=plugins=grpc:. ./proto/sfapi.proto
```


