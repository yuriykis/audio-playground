// Command server accepts audio clients over WebSocket, decodes their Opus
// packets and saves every connection as a WAV recording.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"audio-playground/internal/audio"

	"github.com/coder/websocket"
	"gopkg.in/hraban/opus.v2"
)

const (
	defaultAddr   = ":9000"
	recordingsDir = "recordings"

	// maxDecodedSamples fits the longest Opus packet, 120 ms. Our client sends 20 ms.
	maxDecodedSamples = audio.SampleRate * 120 / 1000 * audio.Channels
)

func main() {
	addr := flag.String("addr", defaultAddr, "address to listen on")
	flag.Parse()

	if err := os.MkdirAll(recordingsDir, 0o755); err != nil {
		slog.Error("create recordings directory", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /audio", handleAudio)

	slog.Info("listening", "addr", *addr, "endpoint", "/audio", "recordings", recordingsDir)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

// handleAudio serves one client connection: it decodes every packet back to
// PCM and appends it to a WAV file named after the connection time.
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

	path := filepath.Join(recordingsDir, time.Now().Format("2006-01-02_15-04-05.000")+".wav")
	recording, err := createWAV(path)
	if err != nil {
		log.Error("start recording", "err", err)
		return
	}
	defer func() {
		if err := recording.Close(); err != nil {
			log.Error("save recording", "path", path, "err", err)
			return
		}
		log.Info("recording saved", "path", path, "duration", recording.Duration())
	}()

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
		if err := recording.Write(pcm[:n*audio.Channels]); err != nil {
			log.Error("write recording", "err", err)
			return
		}
		log.Info("received packet", "bytes", len(packet), "samples", n)
	}
}
