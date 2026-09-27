# syntax=docker/dockerfile:1

# The web and build stages run on the build machine and cross-compile, so a
# multi-arch build does not emulate npm or the Go compiler.
FROM --platform=$BUILDPLATFORM node:20-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.23-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY cmd/ cmd/
COPY internal/ internal/
COPY --from=web /src/web/dist/ internal/webembed/dist/
RUN GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags "-s -w" -o /out/craftsail-growth ./cmd/craftsail-growth

FROM alpine:3.20
LABEL org.opencontainers.image.source="https://github.com/craftsail/craftsail-growth" \
      org.opencontainers.image.description="Self-hosted GEO and AI SEO dashboard" \
      org.opencontainers.image.licenses="AGPL-3.0-or-later"
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
