FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/duku-worker ./cmd/worker

FROM alpine:3.21
RUN apk add --no-cache tshark
COPY --from=build /out/duku-worker /usr/local/bin/duku-worker
ENTRYPOINT ["duku-worker"]
