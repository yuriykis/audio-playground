package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gen2brain/malgo"
	"gopkg.in/hraban/opus.v2"
)

const (
	sampleRate = 48000 // samples per second
	channels   = 1     // mono

	// Opus only accepts frames of 2.5, 5, 10, 20, 40 or 60 ms; 20 ms is the usual choice.
	opusFrameSamples = sampleRate / 50 * channels // 20 ms = 960 samples
	maxPacketBytes   = 4000                       // upper bound for one encoded Opus packet
)

// encoder collects microphone samples and encodes them into Opus packets.
type encoder struct {
	opus    *opus.Encoder
	pending []int16 // samples waiting until a full Opus frame is collected
	packet  []byte  // reused output buffer for one encoded packet
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

func main() {
	enc, err := opus.NewEncoder(sampleRate, channels, opus.AppVoIP)
	if err != nil {
		log.Fatal(err)
	}
	e := &encoder{opus: enc, packet: make([]byte, maxPacketBytes)}

	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = ctx.Uninit()
		ctx.Free()
	}()

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16 // each sample is an int16 (2 bytes)
	cfg.Capture.Channels = channels
	cfg.SampleRate = sampleRate

	device, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: e.onData})
	if err != nil {
		log.Fatal(err)
	}
	defer device.Uninit()

	if err := device.Start(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Listening to the microphone... Press Enter to stop.")
	_, _ = fmt.Scanln()
}
