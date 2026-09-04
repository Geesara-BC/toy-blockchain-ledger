# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/blockchain-node ./cmd/node

FROM alpine:3.22

RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
COPY --from=build /out/blockchain-node /app/blockchain-node
RUN mkdir -p /app/data/node1 /app/data/node2 /app/data/node3 && chown -R app:app /app
USER app

EXPOSE 8081 8082 8083
ENTRYPOINT ["/app/blockchain-node"]
