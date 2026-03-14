ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/settlement ./cmd/settlement

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/settlement /bin/settlement
CMD ["/bin/settlement"]
