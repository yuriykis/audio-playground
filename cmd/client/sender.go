package main

import (
	"context"
	"fmt"
	"log/slog"

	"audio-playground/internal/audio"

	"github.com/coder/websocket"
)

// send writes each packet to the server as one binary WebSocket message,
// until packets is closed or a write fails.
func send(ctx context.Context, conn *websocket.Conn, packets <-chan packet) error {
	for p := range packets {
		if err := conn.Write(ctx, websocket.MessageBinary, p.data); err != nil {
			return fmt.Errorf("send packet: %w", err)
		}
		slog.Info("sent packet",
			"pcm_bytes", audio.FrameBytes,
			"opus_bytes", len(p.data),
			"ratio", audio.FrameBytes/len(p.data),
			"peak", p.peak)
	}
	return nil
}
