.PHONY: all test clean
.DEFAULT_GOAL := all

all:

GENERATED_PROTOS=server/fake_server.pb.go httpgrpc/httpgrpc.pb.go middleware/middleware_test/echo_server.pb.go

# All the boiler plate for building golang follows:
SUDO := $(shell docker info >/dev/null 2>&1 || echo "sudo -E")
RM := --rm
GO_FLAGS := -ldflags "-extldflags \"-static\" -linkmode=external -s -w" -tags netgo -i

# A 3-year-old image which is a reasonable match for the last time the files were generated.
PROTOC_IMAGE=namely/protoc:1.22_1

protos:
	docker run $(RM) --user $(id -u):$(id -g) -v $(shell pwd):/go/src/github.com/weaveworks/common -w /go/src/github.com/weaveworks/common namely/protoc:1.22_1 --proto_path=/go/src/github.com/weaveworks/common --go_out=plugins=grpc:/go/src/ server/fake_server.proto
	docker run $(RM) --user $(id -u):$(id -g) -v $(shell pwd):/go/src/github.com/weaveworks/common -w /go/src/github.com/weaveworks/common namely/protoc:1.22_1 --proto_path=/go/src/github.com/weaveworks/common --go_out=plugins=grpc:/go/src/ middleware/middleware_test/echo_server.proto
	docker run $(RM) --user $(id -u):$(id -g) -v $(shell pwd):/go/src/github.com/weaveworks/common -w /go/src/github.com/weaveworks/common namely/protoc:1.22_1 --proto_path=/go/src/github.com/weaveworks/common --gogofast_out=plugins=grpc:/go/src/ httpgrpc/httpgrpc.proto

protos: $(GENERATED_PROTOS)

%.pb.go: %.proto
	protoc --go_out=plugins=grpc:../../.. $<

lint:
	golangci-lint run --new-from-rev d2f56921e6b0

test:
	go test ./...

clean:
	go clean ./...
