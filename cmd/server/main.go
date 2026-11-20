// Command server accepts audio clients over WebSocket and logs every Opus
// packet it receives.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"

	"github.com/coder/websocket"
)

const defaultAddr = ":9000"

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

// handleAudio serves one client connection and logs every packet it receives.
func handleAudio(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		slog.Warn("accept websocket", "err", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	log := slog.With("client", r.RemoteAddr)
	log.Info("client connected")

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
		log.Info("received packet", "bytes", len(packet))
	}
}
