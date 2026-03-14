# Build (GO_VERSION from .go-version via build-arg in CI)
ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
ENV GOPROXY=https://proxy.golang.org,direct
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

# Run
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/api /bin/api
EXPOSE 8080
CMD ["/bin/api"]
