FROM golang:1.24-alpine AS build
WORKDIR /src
COPY server/go.mod server/go.sum ./
RUN go mod download
COPY server/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/snakerule-server ./cmd/main
COPY pulse-agent/ /tmp/pulse-agent/
ARG TARGETARCH
RUN grep " pulse-agent-linux-${TARGETARCH}$" /tmp/pulse-agent/pulse-agent.sha256 | (cd /tmp/pulse-agent && sha256sum -c -) \
    && cp "/tmp/pulse-agent/pulse-agent-linux-${TARGETARCH}" /out/pulse-agent

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S -g 10001 snakerule \
    && adduser -S -D -H -u 10001 -G snakerule snakerule \
    && mkdir -p /run/pulse /opt/pulse/bin \
    && chown -R 10001:10001 /run/pulse /opt/pulse
COPY --from=build --chown=10001:10001 /out/snakerule-server /usr/local/bin/snakerule-server
COPY --from=build --chown=10001:10001 /out/pulse-agent /opt/pulse/bin/pulse-agent
USER 10001:10001
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/snakerule-server"]
