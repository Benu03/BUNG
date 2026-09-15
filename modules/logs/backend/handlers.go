package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
)

type api struct {
	docker *dockerClient
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (a *api) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "module": "logs"})
}

func (a *api) listContainers(w http.ResponseWriter, r *http.Request) {
	containers, err := a.docker.ListContainers(r.Context())
	if err != nil {
		log.Printf("list containers failed: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to list containers"})
		return
	}
	writeJSON(w, http.StatusOK, containers)
}

// streamLogs is a Server-Sent Events endpoint - one `data: <line>` event
// per log line, streamed live as Docker produces them. The frontend uses
// the browser's native EventSource, which handles reconnection itself.
func (a *api) streamLogs(w http.ResponseWriter, r *http.Request) {
	containerID := r.PathValue("id")
	tail := 200
	if v := r.URL.Query().Get("tail"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 2000 {
			tail = n
		}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	err := a.docker.StreamLogs(ctx, containerID, tail, func(line string) {
		// demux already guarantees `line` has no embedded newline, which
		// is the one hard requirement for a single SSE `data:` field.
		fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	})
	if err != nil {
		log.Printf("stream logs for %s: %v", containerID, err)
	}
}
