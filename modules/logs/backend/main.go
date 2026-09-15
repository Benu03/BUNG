package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	sockPath := getenv("DOCKER_SOCK", "/var/run/docker.sock")

	a := &api{docker: newDockerClient(sockPath)}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /containers", a.listContainers)
	mux.HandleFunc("GET /containers/{id}/stream", a.streamLogs)

	log.Printf("logs-backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
