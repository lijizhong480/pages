FROM golang:1.23-alpine AS build

WORKDIR /src
COPY go.mod ./
COPY main.go ./
COPY web ./web
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/page-service .

FROM alpine:3.21

RUN apk add --no-cache ca-certificates \
    && mkdir -p /app/data
WORKDIR /app
COPY --from=build /out/page-service /usr/local/bin/page-service

ENV PAGE_ADDR=:8080 \
    PAGE_DATA=/app/data

EXPOSE 8080
VOLUME ["/app/data"]
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/usr/local/bin/page-service"]
