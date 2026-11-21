// Command server accepts audio clients over WebSocket and decodes every Opus
// packet it receives back to PCM.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"audio-playground/internal/audio"

	"github.com/coder/websocket"
	"gopkg.in/hraban/opus.v2"
)

const (
	defaultAddr = ":9000"

	// maxDecodedSamples fits the longest Opus packet, 120 ms. Our client sends 20 ms.
	maxDecodedSamples = audio.SampleRate * 120 / 1000 * audio.Channels
)

func main() {
	addr := flag.String("addr", defaultAddr, "address to listen on")
	flag.Parse()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /audio", handleAudio)

	slog.Info("listening", "addr", *addr, "endpoint", "/audio")
	if err := http.ListenAndServe(*addr, mux); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// handleAudio serves one client connection and decodes every packet back to PCM.
func handleAudio(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("accept websocket", "err", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	log := slog.With("client", r.RemoteAddr)
	log.Info("client connected")

	dec, err := opus.NewDecoder(audio.SampleRate, audio.Channels)
	if err != nil {
		log.Error("create decoder", "err", err)
		return
	}

	pcm := make([]int16, maxDecodedSamples)
	for {
		_, packet, err := conn.Read(r.Context())
		if websocket.CloseStatus(err) == websocket.StatusNormalClosure {
			log.Info("client disconnected")
			return
		}
		if err != nil {
			log.Warn("read packet", "err", err)
			return
		}

		// Decode returns the number of samples per channel.
		n, err := dec.Decode(packet, pcm)
		if err != nil {
			log.Warn("decode packet", "err", err)
			continue
		}
		log.Info("received packet", "bytes", len(packet), "samples", n)
	}
}
