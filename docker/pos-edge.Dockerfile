FROM golang:1.26-alpine AS build

WORKDIR /src
COPY shared/platform/go.mod ./shared/platform/go.mod
COPY pos-backend/go.mod pos-backend/go.sum ./pos-backend/

WORKDIR /src/pos-backend
RUN go mod download

WORKDIR /src
COPY shared/platform ./shared/platform
COPY pos-backend ./pos-backend

WORKDIR /src/pos-backend
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pos-edge ./cmd/pos-edge

FROM alpine:3.22

WORKDIR /app
RUN apk add --no-cache tzdata && adduser -D -H pos
COPY --from=build /out/pos-edge /app/pos-edge
COPY --from=build /src/pos-backend/migrations /app/migrations
COPY --from=build /src/pos-backend/config /app/config
RUN mkdir -p /app/data && chown -R pos:pos /app

USER pos
EXPOSE 8080
ENV POS_CONFIG_PATH=/app/config/pos-edge.docker.json
CMD ["/app/pos-edge"]
