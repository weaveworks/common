.PHONY: all test clean
.DEFAULT_GOAL := all

all:

GENERATED_PROTOS=server/fake_server.pb.go httpgrpc/httpgrpc.pb.go middleware/middleware_test/echo_server.pb.go

# All the boiler plate for building golang follows:
SUDO := $(shell docker info >/dev/null 2>&1 || echo "sudo -E")
RM := --rm
GO_FLAGS := -ldflags "-extldflags \"-static\" -linkmode=external -s -w" -tags netgo -i

protos:
	@mkdir -p $(shell pwd)/.pkg
	$(SUDO) docker run $(RM) -ti \
		-v $(shell pwd)/.pkg:/go/pkg \
		-v $(shell pwd):/go/src/github.com/weaveworks/common \
		-e SRC_PATH=/go/src/github.com/weaveworks/common \
		$(BUILD_IMAGE) make $@

protos: $(GENERATED_PROTOS)

%.pb.go: %.proto
	protoc --go_out=plugins=grpc:../../.. $<

lint:
	golangci-lint run --new-from-rev d2f56921e6b0

test:
	go test ./...

clean:
	go clean ./...
