PROJECT_ROOT    := github.com/seizadi/grpc

# configuration for the protobuf gentool
SRCROOT_ON_HOST      := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
SRCROOT_IN_CONTAINER := /go/src/$(PROJECT_ROOT)
DOCKER_RUNNER        := docker run --rm
DOCKER_RUNNER        += -v $(SRCROOT_ON_HOST):$(SRCROOT_IN_CONTAINER)
DOCKER_GENERATOR     := infoblox/atlas-gentool:latest
GENERATOR            := $(DOCKER_RUNNER) $(DOCKER_GENERATOR)


.PHONY: all
all: protobuf build

.PHONY: protobuf
protobuf:
    docker run -v `pwd`:/defs namely/gen-grpc-gateway -f ./proto/sfapi.proto -s Starfriends

.PHONY: protobuff
protobuff:
	@protoc --go_out=plugins=grpc:. ./proto/sfapi.proto
	@$(GENERATOR) \
	--go_out=plugins=grpc:. \
	--grpc-gateway_out=logtostderr=true:. \
	--gorm_out="engine=postgres:." \
	--swagger_out="atlas_patch=true:." \
	--atlas-query-validate_out=. \
	--atlas-validate_out="." \
	--validate_out="lang=go:." 	$(PROJECT_ROOT)/proto/sfapi.proto

.PHONY: build
build:
	@go build -o ./bin/server ./server.go

.PHONY: clean
clean:
	@rm ./bin/server ./proto/*.go