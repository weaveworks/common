FROM golang:1.10.3-stretch
RUN apt-get update && apt-get install -y file python-requests && \
	rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
RUN go clean -i net && \
	go install -tags netgo std && \
	go install -race -tags netgo std
RUN go get -tags netgo \
		github.com/fzipp/gocyclo \
		github.com/golang/lint/golint \
		github.com/kisielk/errcheck \
		github.com/mjibson/esc \
		github.com/client9/misspell/cmd/misspell \
		mvdan.cc/sh/cmd/shfmt
RUN curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
COPY build.sh /
ENTRYPOINT ["/build.sh"]
