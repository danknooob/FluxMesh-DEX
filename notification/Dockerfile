ARG GO_VERSION=1.25
FROM golang:${GO_VERSION}-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /bin/notification ./cmd/notification

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/notification /bin/notification
EXPOSE 8090
CMD ["/bin/notification"]
