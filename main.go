package main

import (
	"encoding/binary"
	"fmt"
	"log"

	"github.com/gen2brain/malgo"
)

const (
	sampleRate = 48000 // samples per second
	channels   = 1     // mono
)

func main() {
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

	device, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: onData})
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

// onData is called by the system every time a new buffer arrives from the microphone.
func onData(_, input []byte, frameCount uint32) {
	samples := make([]int16, frameCount*channels)
	peak := 0
	for i := range samples {
		s := int16(binary.LittleEndian.Uint16(input[i*2:]))
		samples[i] = s
		if abs := max(int(s), -int(s)); abs > peak {
			peak = abs
		}
	}

	ms := float64(frameCount) / sampleRate * 1000
	fmt.Printf("%4d samples (%5.1f ms, %4d bytes)  first: %v  peak: %5d\n",
		len(samples), ms, len(input), samples[:min(5, len(samples))], peak)
}
