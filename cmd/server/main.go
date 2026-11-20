package main

import (
	"errors"
	"log"
	"net/http"

	"github.com/coder/websocket"
)

const addr = ":9000"

func main() {
	http.HandleFunc("/audio", handleAudio)

	log.Printf("Listening on %s (WebSocket endpoint: ws://localhost%s/audio)", addr, addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

// handleAudio accepts one client connection and logs every Opus packet it receives.
func handleAudio(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		log.Printf("accept: %v", err)
		return
	}
	defer func() { _ = conn.CloseNow() }()

	log.Printf("client connected: %s", r.RemoteAddr)

	for {
		_, packet, err := conn.Read(r.Context())
		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure || errors.Is(err, r.Context().Err()) {
				log.Printf("client disconnected: %s", r.RemoteAddr)
			} else {
				log.Printf("read: %v", err)
			}
			return
		}
		log.Printf("received %3d bytes", len(packet))
	}
}
