FROM node:22-alpine AS frontend
WORKDIR /app/web
RUN corepack enable
COPY web/package.json web/yarn.lock web/.yarnrc.yml ./
RUN yarn install --immutable
COPY web/ ./
RUN yarn check && yarn build

FROM golang:1.26.4-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go test ./... && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /server .

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && mkdir /data /app && chown 65532:65532 /data /app
WORKDIR /app
COPY --from=backend /server /app/server
COPY --from=frontend /app/web/build /app/web
ENV LISTEN_ADDR=:8080 DB_PATH=/data/aprs.sqlite WEB_DIR=/app/web TZ=Asia/Shanghai
USER 65532:65532
EXPOSE 8080
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 CMD ["/app/server", "healthcheck"]
ENTRYPOINT ["/app/server"]
