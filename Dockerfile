# syntax=docker/dockerfile:1

FROM node:20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.23-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=web /src/web/dist/ internal/webembed/dist/
RUN go build -trimpath -ldflags "-s -w" -o /out/craftsail-growth ./cmd/craftsail-growth

FROM alpine:3.24
RUN apk add --no-cache ca-certificates tzdata \
 && adduser -D -H -u 10001 app \
 && mkdir -p /app/config /app/data \
 && chown -R app:app /app
WORKDIR /app
COPY --from=build /out/craftsail-growth /usr/local/bin/craftsail-growth
COPY --chown=app:app config/default.example.toml config/default.example.toml
USER app
ENV CRAFTSAIL_GROWTH_HOST=0.0.0.0
EXPOSE 8765
VOLUME ["/app/config", "/app/data"]
ENTRYPOINT ["craftsail-growth"]
CMD ["server"]
