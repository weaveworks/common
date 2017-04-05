FROM golang:1.8.0-stretch
RUN apt-get update && apt-get install -y python-requests python-yaml file jq unzip protobuf-compiler libprotobuf-dev && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN go clean -i net && \
	go install -tags netgo std && \
	go install -race -tags netgo std
COPY .src/ /go/src/github.com/weaveworks/common
RUN go get -tags netgo -t github.com/weaveworks/common/... && \
	rm -rf /go/src/github.com/weaveworks/common
RUN go get -tags netgo \
		github.com/golang/dep/... \
		github.com/FiloSottile/gvt \
		github.com/fzipp/gocyclo \
		github.com/golang/lint/golint \
		github.com/kisielk/errcheck \
		github.com/mjibson/esc \
		github.com/client9/misspell/cmd/misspell \
		github.com/mvdan/sh/cmd/shfmt \
		github.com/fatih/hclfmt \
		github.com/golang/protobuf/protoc-gen-go \
		github.com/gogo/protobuf/protoc-gen-gogoslick \
		github.com/gogo/protobuf/gogoproto && \
	rm -rf /go/pkg /go/src
COPY build.sh /
ENTRYPOINT ["/build.sh"]
