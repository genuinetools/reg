# Set an output prefix, which is the local directory if not specified
PREFIX?=$(shell pwd)
BUILDTAGS=

.PHONY: clean all fmt vet lint build test install static
.DEFAULT: default

all: clean build fmt lint test vet install

build:
	@echo "+ $@"
	@go build -tags "$(BUILDTAGS) cgo" .

static:
	@echo "+ $@"
	CGO_ENABLED=1 go build -tags "$(BUILDTAGS) cgo static_build" -ldflags "-w -extldflags -static" -o reg .

fmt:
	@echo "+ $@"
	@gofmt -s -l . | grep -v vendor | tee /dev/stderr

lint:
	@echo "+ $@"
	@golint ./... | grep -v vendor | tee /dev/stderr

test:
	@echo "+ $@"
	@go get github.com/labstack/echo
	@go get github.com/labstack/echo/middleware
	@go test -v -tags "$(BUILDTAGS) cgo" $(shell go list ./... | grep -v vendor)

vet:
	@echo "+ $@"
	@go vet $(shell go list ./... | grep -v vendor)

clean:
	@echo "+ $@"
	@rm -rf reg
	@rm -rf $(CURDIR)/.certs

install:
	@echo "+ $@"
	@go install .

# set the graph driver as the current graphdriver if not set
DOCKER_GRAPHDRIVER := $(if $(DOCKER_GRAPHDRIVER),$(DOCKER_GRAPHDRIVER),$(shell docker info 2>&1 | grep "Storage Driver" | sed 's/.*: //'))
export DOCKER_GRAPHDRIVER

# if this session isn't interactive, then we don't want to allocate a
# TTY, which would fail, but if it is interactive, we do want to attach
# so that the user can send e.g. ^C through.
INTERACTIVE := $(shell [ -t 0 ] && echo 1 || echo 0)
ifeq ($(INTERACTIVE), 1)
	DOCKER_FLAGS += -t
endif

.PHONY: dind
DIND_CONTAINER=reg-dind
DIND_DOCKER_IMAGE=r.j3ss.co/docker:userns
dind:
	docker build --rm --force-rm -f Dockerfile.dind -t $(DIND_DOCKER_IMAGE) .
	docker run -d  \
		-v /var/lib/docker2:/var/lib/docker \
		--name $(DIND_CONTAINER) \
		--privileged \
		-v $(CURDIR)/.certs:/etc/docker/ssl \
		-v $(CURDIR):/go/src/github.com/jessfraz/reg \
		-v /tmp:/tmp \
		$(DIND_DOCKER_IMAGE) \
		docker daemon -D --storage-driver $(DOCKER_GRAPHDRIVER) \
		-H tcp://127.0.0.1:2375 \
		--host=unix:///var/run/docker.sock \
		--disable-legacy-registry=true \
		--exec-opt=native.cgroupdriver=cgroupfs \
		--insecure-registry localhost:5000 \
		--tlsverify \
		--tlscacert=/etc/docker/ssl/ca.pem \
		--tlskey=/etc/docker/ssl/key.pem \
		--tlscert=/etc/docker/ssl/cert.pem

.PHONY: dtest
DOCKER_IMAGE := reg-dev
dtest:
	docker build --rm --force-rm -f Dockerfile.dev -t $(DOCKER_IMAGE) .
	docker run --rm -i $(DOCKER_FLAGS) \
		-v $(CURDIR):/go/src/github.com/jessfraz/reg \
		--workdir /go/src/github.com/jessfraz/reg \
		-v $(CURDIR)/.certs:/etc/docker/ssl:ro \
		-v /tmp:/tmp \
		--net container:$(DIND_CONTAINER) \
		-e DOCKER_HOST=tcp://127.0.0.1:2375 \
		-e DOCKER_TLS_VERIFY=true \
		-e DOCKER_CERT_PATH=/etc/docker/ssl \
		-e DOCKER_API_VERSION=1.23 \
		$(DOCKER_IMAGE) \
		make test
