# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- run stage ----
FROM alpine:3.20
RUN adduser -D -H appuser && mkdir -p /data && chown appuser:appuser /data
WORKDIR /
COPY --from=build /app/server /server
USER appuser
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/server"]
