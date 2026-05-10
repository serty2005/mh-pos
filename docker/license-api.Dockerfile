FROM golang:1.26-alpine AS build

WORKDIR /src
COPY shared/platform/go.mod ./shared/platform/go.mod
COPY license-server/go.mod license-server/go.sum ./license-server/

WORKDIR /src/license-server
RUN go mod download

WORKDIR /src
COPY shared/platform ./shared/platform
COPY license-server ./license-server

WORKDIR /src/license-server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/license-api ./cmd/license-api

FROM alpine:3.22

WORKDIR /app
RUN adduser -D -H license
COPY --from=build /out/license-api /app/license-api
COPY --from=build /src/license-server/config /app/config
RUN mkdir -p /app/data && chown -R license:license /app

USER license
EXPOSE 8095
ENV LICENSE_CONFIG_PATH=/app/config/license-api.docker.json
CMD ["/app/license-api"]
