FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/pos-edge ./cmd/pos-edge

FROM alpine:3.22
WORKDIR /app
RUN apk add --no-cache tzdata && adduser -D -H pos
COPY --from=build /out/pos-edge /app/pos-edge
COPY migrations /app/migrations
RUN mkdir -p /app/data && chown -R pos:pos /app
USER pos
EXPOSE 8080
ENV POS_HTTP_ADDR=:8080
ENV POS_SQLITE_PATH=/app/data/pos-edge.db
ENV POS_SQLITE_MIGRATIONS_DIR=/app/migrations/sqlite
CMD ["/app/pos-edge"]
