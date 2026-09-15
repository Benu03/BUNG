# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- run stage ----
FROM alpine:3.20
WORKDIR /
COPY --from=build /app/server /server
# Deliberately runs as root - the one exception in this project. Reading
# /var/run/docker.sock (mounted read-only, see docker-compose.yml)
# generally requires it on a standard Docker host. See docker.go's
# SECURITY NOTE for the full tradeoff this accepts.
EXPOSE 8080
ENTRYPOINT ["/server"]
