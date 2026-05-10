FROM golang:1.26-alpine AS build

WORKDIR /src
COPY shared/platform/go.mod ./shared/platform/go.mod
COPY cloud-backend/go.mod cloud-backend/go.sum ./cloud-backend/

WORKDIR /src/cloud-backend
RUN go mod download

WORKDIR /src
COPY shared/platform ./shared/platform
COPY cloud-backend ./cloud-backend

WORKDIR /src/cloud-backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/cloud-api ./cmd/cloud-api

FROM alpine:3.22

WORKDIR /app
RUN adduser -D -H cloud
COPY --from=build /out/cloud-api /app/cloud-api
COPY --from=build /src/cloud-backend/migrations /app/migrations
COPY --from=build /src/cloud-backend/config /app/config
RUN mkdir -p /app/data/cloud-backups && chown -R cloud:cloud /app

USER cloud
EXPOSE 8090
ENV CLOUD_CONFIG_PATH=/app/config/cloud-api.docker.json
CMD ["/app/cloud-api"]
