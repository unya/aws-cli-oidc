FROM golang:alpine AS build_base

RUN apk add --no-cache git
RUN apk add --no-cache make

# Set the Current Working Directory inside the container
WORKDIR /tmp/aws-cli-oidc

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN make build

# Start fresh from a smaller image
FROM alpine:latest

COPY --from=build_base /tmp/aws-cli-oidc/bin/aws-cli-oidc /app/aws-cli-oidc

ENV user app_user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser
RUN mkdir -p ~/.aws-cli-oidc

# This container exposes port 8080 to the outside world
EXPOSE 52327

# Run the binary program produced by `go install`
# CMD ["/app/aws-cli-oidc"]
ENTRYPOINT ["/app/aws-cli-oidc"]
# CMD ["bash"]