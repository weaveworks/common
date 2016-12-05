FROM golang:1.7.4
ENV SRC_NAME github.com/weaveworks/common
ENV SRC_PATH /go/src/${SRC_NAME}
RUN apt-get update && apt-get install -y file && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN go clean -i net && \
	go install -tags netgo std && \
	go install -race -tags netgo std
COPY .src/ ${SRC_PATH}
RUN go get -tags netgo -t ${SRC_NAME}/... && rm -rf ${SRC_PATH}
RUN go get -tags netgo \
		github.com/fzipp/gocyclo \
		github.com/golang/lint/golint \
		github.com/kisielk/errcheck \
		github.com/mjibson/esc \
		github.com/client9/misspell/cmd/misspell \
        github.com/mvdan/sh/cmd/shfmt
COPY build.sh /
ENTRYPOINT ["/build.sh"]
