package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sync"

	"github.com/coder/websocket"
	"github.com/gen2brain/malgo"
	"gopkg.in/hraban/opus.v2"
)

const (
	sampleRate = 48000 // samples per second
	channels   = 1     // mono

	// Opus only accepts frames of 2.5, 5, 10, 20, 40 or 60 ms; 20 ms is the usual choice.
	opusFrameSamples = sampleRate / 50 * channels // 20 ms = 960 samples
	maxPacketBytes   = 4000                       // upper bound for one encoded Opus packet

	serverURL     = "ws://localhost:9000/audio"
	sendQueueSize = 50 // packets waiting to be sent; 50 x 20 ms = 1 s of audio
)

// encoder collects microphone samples and encodes them into Opus packets.
type encoder struct {
	opus    *opus.Encoder
	pending []int16       // samples waiting until a full Opus frame is collected
	packet  []byte        // reused output buffer for one encoded packet
	out     chan<- []byte // encoded packets handed over to the sender goroutine
}

// onData is called by the system every time a new buffer arrives from the microphone.
func (e *encoder) onData(_, input []byte, frameCount uint32) {
	for i := range int(frameCount) * channels {
		e.pending = append(e.pending, int16(binary.LittleEndian.Uint16(input[i*2:])))
	}

	for len(e.pending) >= opusFrameSamples {
		frame := e.pending[:opusFrameSamples]

		n, err := e.opus.Encode(frame, e.packet)
		if err != nil {
			log.Printf("opus encode: %v", err)
		} else {
			pcmBytes := len(frame) * 2
			fmt.Printf("%d samples (%d bytes PCM) -> %3d bytes Opus  (%2dx smaller)  peak: %5d\n",
				len(frame), pcmBytes, n, pcmBytes/n, peak(frame))

			// Copy the packet, because e.packet is overwritten by the next Encode.
			// Never block the audio thread: if the queue is full, drop the packet.
			select {
			case e.out <- append([]byte(nil), e.packet[:n]...):
			default:
				log.Printf("send queue full, dropped a %d-byte packet", n)
			}
		}

		// Drop the encoded frame and keep the leftover samples for the next one.
		e.pending = append(e.pending[:0], e.pending[opusFrameSamples:]...)
	}
}

// peak returns the loudest absolute sample value.
func peak(samples []int16) int {
	p := 0
	for _, s := range samples {
		p = max(p, int(s), -int(s))
	}
	return p
}

// send writes every queued packet to the server as one binary WebSocket message.
func send(ctx context.Context, conn *websocket.Conn, packets <-chan []byte) {
	for packet := range packets {
		if err := conn.Write(ctx, websocket.MessageBinary, packet); err != nil {
			log.Printf("send: %v", err)
			return
		}
	}
}

func main() {
	ctx := context.Background()

	conn, _, err := websocket.Dial(ctx, serverURL, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = conn.CloseNow() }()

	packets := make(chan []byte, sendQueueSize)
	var sender sync.WaitGroup
	sender.Go(func() { send(ctx, conn, packets) })

	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		log.Fatal(err)
	}
	e := &encoder{opus: enc, packet: make([]byte, maxPacketBytes), out: packets}

	audioCtx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = audioCtx.Uninit()
		audioCtx.Free()
	}()

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16 // each sample is an int16 (2 bytes)
	cfg.Capture.Channels = channels
	cfg.SampleRate = sampleRate

	device, err := malgo.InitDevice(audioCtx.Context, cfg, malgo.DeviceCallbacks{Data: e.onData})
	if err != nil {
		log.Fatal(err)
	}

	if err := device.Start(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Streaming the microphone to %s... Press Enter to stop.\n", serverURL)
	_, _ = fmt.Scanln()

	// Shut down in order: stop the microphone (no more packets), let the sender
	// flush the queue, then close the connection cleanly.
	device.Uninit()
	close(packets)
	sender.Wait()
	_ = conn.Close(websocket.StatusNormalClosure, "")
}
