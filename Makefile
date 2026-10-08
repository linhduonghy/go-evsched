OS ?= linux
ARCH ?= amd64
OUTPUT_DIR ?= bin
APP_NAME ?= evsched

.PHONY: build clean docker docker-build

build: build

build:
	OS=$(OS) ARCH=$(ARCH) OUTPUT_DIR=$(OUTPUT_DIR) APP_NAME=$(APP_NAME) ./build.sh

clean:
	rm -rf $(OUTPUT_DIR)