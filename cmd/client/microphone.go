package main

import (
	"encoding/binary"
	"fmt"

	"audio-playground/internal/audio"

	"github.com/gen2brain/malgo"
)

// microphone captures 16-bit PCM from the default input device.
type microphone struct {
	ctx     *malgo.AllocatedContext
	device  *malgo.Device
	samples []int16 // reused for every buffer, so the audio thread does not allocate
}

// openMicrophone prepares the default input device. onSamples is called on the
// audio thread for every buffer (about every 10 ms); the slice it receives is
// only valid during that call and onSamples must never block.
func openMicrophone(onSamples func([]int16)) (*microphone, error) {
	ctx, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, fmt.Errorf("init audio context: %w", err)
	}
	m := &microphone{ctx: ctx}

	cfg := malgo.DefaultDeviceConfig(malgo.Capture)
	cfg.Capture.Format = malgo.FormatS16 // each sample is an int16 (2 bytes)
	cfg.Capture.Channels = audio.Channels
	cfg.SampleRate = audio.SampleRate

	onData := func(_, input []byte, _ uint32) {
		m.samples = decodePCM(m.samples[:0], input)
		onSamples(m.samples)
	}

	device, err := malgo.InitDevice(ctx.Context, cfg, malgo.DeviceCallbacks{Data: onData})
	if err != nil {
		m.Close()
		return nil, fmt.Errorf("open input device: %w", err)
	}
	m.device = device
	return m, nil
}

// Start begins capturing; from now on onSamples is called.
func (m *microphone) Start() error {
	if err := m.device.Start(); err != nil {
		return fmt.Errorf("start input device: %w", err)
	}
	return nil
}

// Close stops capturing and releases the device. After it returns, onSamples
// is no longer called. It is safe to call Close more than once.
func (m *microphone) Close() {
	if m.device != nil {
		m.device.Uninit()
		m.device = nil
	}
	if m.ctx != nil {
		_ = m.ctx.Uninit()
		m.ctx.Free()
		m.ctx = nil
	}
}

// decodePCM appends the little-endian 16-bit samples from raw to dst.
func decodePCM(dst []int16, raw []byte) []int16 {
	for i := 0; i+audio.BytesPerSample <= len(raw); i += audio.BytesPerSample {
		dst = append(dst, int16(binary.LittleEndian.Uint16(raw[i:])))
	}
	return dst
}
