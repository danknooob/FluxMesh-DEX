# Build
FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/api ./cmd/api

# Run
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/api /bin/api
EXPOSE 8080
CMD ["/bin/api"]
