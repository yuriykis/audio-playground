package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"sync/atomic"

	"audio-playground/internal/audio"

	"gopkg.in/hraban/opus.v2"
)

// packet is one encoded Opus frame ready to be sent.
type packet struct {
	data []byte
	peak int // loudest sample in the frame, only for logging
}

// encoder groups microphone samples into Opus frames, encodes them and
// queues the packets for the sender.
type encoder struct {
	opus    *opus.Encoder
	pending []int16       // samples waiting until a full Opus frame is collected
	buf     []byte        // reused output buffer for one encoded packet
	out     chan<- packet // packets handed over to the sender
	dropped atomic.Int64  // packets dropped because the queue was full
}

func newEncoder(out chan<- packet) (*encoder, error) {
	enc, err := opus.NewEncoder(audio.SampleRate, audio.Channels, opus.AppVoIP)
	if err != nil {
		return nil, fmt.Errorf("create opus encoder: %w", err)
	}
	return &encoder{opus: enc, buf: make([]byte, audio.MaxPacketBytes), out: out}, nil
}

// write is called on the audio thread with every microphone buffer, so it
// must never block.
func (e *encoder) write(samples []int16) {
	e.pending = append(e.pending, samples...)

	for len(e.pending) >= audio.FrameSamples {
		e.encode(e.pending[:audio.FrameSamples])

		// Drop the encoded frame and keep the leftover samples for the next one.
		e.pending = append(e.pending[:0], e.pending[audio.FrameSamples:]...)
	}
}

func (e *encoder) encode(frame []int16) {
	n, err := e.opus.Encode(frame, e.buf)
	if err != nil {
		slog.Error("encode opus frame", "err", err)
		return
	}

	// Copy the packet, because e.buf is overwritten by the next Encode.
	p := packet{data: bytes.Clone(e.buf[:n]), peak: peak(frame)}

	// If the sender cannot keep up, drop the packet instead of waiting.
	select {
	case e.out <- p:
	default:
		e.dropped.Add(1)
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
