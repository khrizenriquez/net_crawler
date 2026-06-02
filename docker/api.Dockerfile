FROM golang:1.23-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY go.sum ./
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -o /out/duku-api ./cmd/api

FROM alpine:3.21
COPY --from=build /out/duku-api /usr/local/bin/duku-api
USER nobody
ENTRYPOINT ["duku-api"]
