package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

// dockerClient talks to the Docker Engine API over the Unix socket
// mounted read-only into this container (see docker-compose.yml) - no
// SDK dependency, just a plain http.Client with a custom dialer, since
// the API surface this needs (list containers, stream one container's
// logs) is small enough to hand-roll against stdlib alone.
//
// SECURITY NOTE (read before touching this file): a "read-only" bind
// mount of docker.sock only stops this container from replacing the
// socket file itself - once connected, the full Docker Engine API
// (including creating/starting/deleting containers, which is
// root-equivalent host access) is reachable over that connection. This
// service deliberately only ever issues GET requests below (list
// containers, stream logs) - never anything that creates, starts, or
// deletes - but the capability to do more exists at the socket level
// regardless of what this code does. That's why this feature is gated
// behind app-maintenance's own admin-tier module grant (see
// /modules/logs/nginx.conf) rather than being open to everyone, and why
// its container (see Dockerfile) is the one exception in this whole
// project that isn't dropped to a non-root user - reading the socket
// generally requires it. A deliberate, documented tradeoff, not an
// oversight.
type dockerClient struct {
	http *http.Client
}

func newDockerClient(sockPath string) *dockerClient {
	return &dockerClient{
		http: &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
					var d net.Dialer
					return d.DialContext(ctx, "unix", sockPath)
				},
			},
		},
	}
}

type containerInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Image string `json:"image"`
	State string `json:"state"`
}

// rawContainer mirrors just the fields this needs from Docker's
// GET /containers/json response.
type rawContainer struct {
	ID     string   `json:"Id"`
	Names  []string `json:"Names"`
	Image  string   `json:"Image"`
	State  string   `json:"State"`
}

func (c *dockerClient) ListContainers(ctx context.Context) ([]containerInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://docker/containers/json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker api: %s", resp.Status)
	}

	var raw []rawContainer
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	out := make([]containerInfo, 0, len(raw))
	for _, rc := range raw {
		name := rc.ID
		if len(rc.Names) > 0 {
			name = strings.TrimPrefix(rc.Names[0], "/")
		}
		out = append(out, containerInfo{ID: rc.ID, Name: name, Image: rc.Image, State: rc.State})
	}
	return out, nil
}

// StreamLogs opens Docker's follow-mode log stream for containerID and
// calls onLine once per complete line (already de-multiplexed - see
// demux) until ctx is cancelled or the stream ends.
func (c *dockerClient) StreamLogs(ctx context.Context, containerID string, tail int, onLine func(line string)) error {
	url := fmt.Sprintf("http://docker/containers/%s/logs?follow=1&stdout=1&stderr=1&timestamps=1&tail=%d", containerID, tail)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker api: %s", resp.Status)
	}
	return demux(resp.Body, onLine)
}

// demux strips Docker's log stream framing - an 8-byte header per frame
// (1 byte stream type, 3 reserved, 4-byte big-endian payload length),
// present because our containers don't allocate a tty - and calls onLine
// once per complete line within each frame's payload.
func demux(r io.Reader, onLine func(line string)) error {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		size := binary.BigEndian.Uint32(header[4:8])
		payload := make([]byte, size)
		if _, err := io.ReadFull(r, payload); err != nil {
			return err
		}
		for _, line := range strings.Split(strings.TrimRight(string(payload), "\n"), "\n") {
			if line != "" {
				onLine(line)
			}
		}
	}
}
