NAME := aws-cli-oidc

SRCS    := $(shell find . -type f -name '*.go')
LDFLAGS := -ldflags="-s -w -extldflags -static"

DIST_DIRS := find * -type d -exec

.DEFAULT_GOAL := bin/$(NAME)

bin/$(NAME): $(SRCS)
	go build $(LDFLAGS) -o bin/$(NAME) cmd/$(NAME)/main.go

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf dist/*
	rm -rf vendor/*

.PHONY: cross-build
cross-build:
	for os in darwin linux windows; do \
	    [ $$os = "windows" ] && EXT=".exe"; \
		for arch in amd64; do \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a -tags netgo -installsuffix netgo $(LDFLAGS) -o dist/$$os-$$arch/$(NAME)$$EXT cmd/$(NAME)/main.go; \
		done; \
	done

.PHONY: deps
deps:
	GO111MODULE=on go mod vendor

.PHONY: dist
dist:
	cd dist && \
	$(DIST_DIRS) cp ../LICENSE {} \; && \
	$(DIST_DIRS) cp ../README.md {} \; && \
	$(DIST_DIRS) tar -zcf $(NAME)-{}.tar.gz {} \; && \
	$(DIST_DIRS) zip -r $(NAME)-{}.zip {} \; && \
	cd ..

.PHONY: install
install:
	go install $(LDFLAGS)

.PHONY: test
test:
	go test -cover -v ./internal/...

