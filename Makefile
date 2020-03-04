NAME := aws-cli-oidc

LDFLAGS := -ldflags="-s -w -extldflags -static"

DIST_DIRS := find * -type d -exec

build:
	go build $(LDFLAGS) -o ./bin/ ./cmd/...

.PHONY: clean
clean:
	rm -rf bin/*
	rm -rf dist/*
	rm -rf vendor/*

.PHONY: cross-build
cross-build:
	for os in darwin linux windows; do \
		for arch in amd64; do \
			mkdir -p ./dist/$$os-$$arch/; \
			GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -a $(LDFLAGS) -o ./dist/$$os-$$arch/ ./cmd/...; \
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

.PHONY: lint
lint:
	golangci-lint run
