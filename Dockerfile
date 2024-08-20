# syntax=docker/dockerfile:1
# docker build . -t airdrop-viewer-api:local
# docker run -p 4001:4001 airdrop-viewer-api:local 4001


FROM golang:1.23 AS build

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download


COPY *.go ./
COPY network/dungeon-1/genesis.json network/dungeon-1/genesis.json

RUN CGO_ENABLED=0 GOOS=linux go build -o /main

FROM alpine:latest AS production

COPY --from=build /main /

ENTRYPOINT ["/main"]
