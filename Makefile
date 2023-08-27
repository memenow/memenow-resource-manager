VERSION := 0.1.0
BUILD_DIR := ./build
DOCKER_IMAGE_NAME := ghcr.io/memenow/memenow-resource-manager
GO_BUILD_CMD := /usr/local/go/bin/go build -o ./build/${DOCKER_IMAGE_NAME}  ./cmd/main.go

.PHONY: build docker

build:
	${GO_BUILD_CMD}

container: build
	docker build -t ${DOCKER_IMAGE_NAME}:${VERSION} ${BUILD_DIR}

container push: container
	docker push ${DOCKER_IMAGE_NAME}:${VERSION}