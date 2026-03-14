FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/notification ./cmd/notification

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/notification /bin/notification
EXPOSE 8090
CMD ["/bin/notification"]
