APP_NAME=bolan

HOST_ARCH := $(shell file `which docker` | awk '{print $$NF}')
ifeq ($(HOST_ARCH), arm64)
	DOCKER_PLATFORM := --platform linux/amd64
endif


.PHONY: build
build: ## Build crawler and service docker images.
	go build -a -o .build/.artifacts/crawler ./cmd/crawler/main.go
	go build -a -o .build/.artifacts/service ./cmd/service/main.go

.PHONY: image
image:
	DOCKER_BUILDKIT=1 docker build $(DOCKER_PLATFORM) -f .build/Dockerfile . -t $(APP_NAME)
