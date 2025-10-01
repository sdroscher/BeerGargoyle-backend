FROM golang:1.24-alpine AS build_base

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /tmp/BeerGargoyle

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Build the Go app
RUN go build -o ./out/BeerGargoyle main.go

# Start fresh from a smaller image
FROM alpine:latest
RUN apk add ca-certificates


COPY --from=build_base /tmp/BeerGargoyle/out/BeerGargoyle /app/BeerGargoyle

# This container exposes port 8080 to the outside world
EXPOSE 8080

# Run the binary program produced by `go install`
CMD ["/app/BeerGargoyle", "serve"]