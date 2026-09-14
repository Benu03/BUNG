# ---- build stage ----
FROM golang:1.23-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server .

# ---- run stage ----
FROM alpine:3.20
RUN adduser -D -H appuser
WORKDIR /
COPY --from=build /app/server /server
USER appuser
EXPOSE 8080
ENTRYPOINT ["/server"]
