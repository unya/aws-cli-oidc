FROM golang:1.13.8

RUN curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.23.6

ENV GOCACHE=/tmp
ENV GOLANGCI_LINT_CACHE=/tmp