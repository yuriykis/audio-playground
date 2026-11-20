// Command client captures the microphone, encodes it with Opus and streams
// the packets to the audio server over WebSocket.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/coder/websocket"
)

const (
	defaultServerURL = "ws://localhost:9000/audio"

	// sendQueueSize is how many packets may wait for the sender: 50 x 20 ms = 1 s of audio.
	sendQueueSize = 50
)

func main() {
	serverURL := flag.String("server", defaultServerURL, "WebSocket URL of the audio server")
	flag.Parse()

	if err := run(*serverURL); err != nil {
		slog.Error("client stopped", "err", err)
		os.Exit(1)
	}
}

func run(serverURL string) error {
	ctx := context.Background()

	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		return fmt.Errorf("connect to %s: %w", serverURL, err)
	}
	defer func() { _ = conn.CloseNow() }()

	packets := make(chan packet, sendQueueSize)
	enc, err := newEncoder(packets)
	if err != nil {
		return err
	}

	mic, err := openMicrophone(enc.write)
	if err != nil {
		return err
	}
	defer mic.Close()

	sendDone := make(chan error, 1)
	go func() { sendDone <- send(ctx, conn, packets) }()

	if err := mic.Start(); err != nil {
		return err
	}
	slog.Info("streaming the microphone, press Enter to stop", "server", serverURL)

	select {
	case <-waitForEnter():
		// Shut down in order: stop the microphone (no more packets),
		// let the sender flush the queue, then close the connection cleanly.
		mic.Close()
		close(packets)
		if err := <-sendDone; err != nil {
			return err
		}
		reportDropped(enc)
		return conn.Close(websocket.StatusNormalClosure, "")

	case err := <-sendDone:
		// The sender stopped on its own, e.g. because the server went away.
		mic.Close()
		reportDropped(enc)
		return err
	}
}

// waitForEnter returns a channel that is closed when the user presses Enter.
// The goroutine cannot be cancelled while it waits on stdin; it ends with the process.
func waitForEnter() <-chan struct{} {
	pressed := make(chan struct{})
	go func() {
		_, _ = fmt.Scanln()
		close(pressed)
	}()
	return pressed
}

func reportDropped(enc *encoder) {
	if n := enc.dropped.Load(); n > 0 {
		slog.Warn("send queue was full, packets dropped", "count", n)
	}
}
