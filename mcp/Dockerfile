FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/mcp ./cmd/mcp

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /bin/mcp /bin/mcp
EXPOSE 8081
CMD ["/bin/mcp"]
