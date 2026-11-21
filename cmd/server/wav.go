package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"audio-playground/internal/audio"
)

// wavWriter saves PCM samples in our format (s16le, mono, 48 kHz) as a WAV file.
//
// A WAV file is a 44-byte header followed by the raw PCM samples. The header
// holds the length of the data, which is unknown until the client disconnects,
// so it is written with zero sizes first and filled in by Close.
type wavWriter struct {
	file      *os.File
	dataBytes uint32
}

// createWAV creates the file at path and writes a placeholder header.
func createWAV(path string) (*wavWriter, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create recording: %w", err)
	}
	w := &wavWriter{file: file}
	if err := w.writeHeader(file); err != nil {
		_ = file.Close()
		return nil, err
	}
	return w, nil
}

// Write appends samples to the end of the file.
func (w *wavWriter) Write(samples []int16) error {
	if err := binary.Write(w.file, binary.LittleEndian, samples); err != nil {
		return fmt.Errorf("write samples: %w", err)
	}
	w.dataBytes += uint32(len(samples) * audio.BytesPerSample)
	return nil
}

// Duration reports how much audio has been written so far.
func (w *wavWriter) Duration() time.Duration {
	samples := int64(w.dataBytes) / audio.BytesPerSample / audio.Channels
	return time.Duration(samples) * time.Second / audio.SampleRate
}

// Close rewrites the header at the start of the file with the real sizes and closes it.
func (w *wavWriter) Close() error {
	headerErr := w.writeHeader(io.NewOffsetWriter(w.file, 0))
	return errors.Join(headerErr, w.file.Close())
}

// writeHeader writes the 44-byte WAV header. All numbers in it are little-endian.
func (w *wavWriter) writeHeader(dst io.Writer) error {
	const bitsPerSample = audio.BytesPerSample * 8

	header := struct {
		RIFF          [4]byte // "RIFF": this is a RIFF container
		RIFFSize      uint32  // bytes in the file after this field
		WAVE          [4]byte // "WAVE": the container holds audio
		Fmt           [4]byte // "fmt ": the format section starts
		FmtSize       uint32  // bytes in the format section: always 16 for PCM
		AudioFormat   uint16  // 1 = uncompressed PCM
		Channels      uint16  // 1 = mono
		SampleRate    uint32  // samples per second per channel
		ByteRate      uint32  // bytes per second
		BlockAlign    uint16  // bytes per frame (one sample from every channel)
		BitsPerSample uint16  // bit depth
		Data          [4]byte // "data": the samples start
		DataSize      uint32  // bytes of samples
	}{
		RIFF:          [4]byte{'R', 'I', 'F', 'F'},
		RIFFSize:      36 + w.dataBytes,
		WAVE:          [4]byte{'W', 'A', 'V', 'E'},
		Fmt:           [4]byte{'f', 'm', 't', ' '},
		FmtSize:       16,
		AudioFormat:   1,
		Channels:      audio.Channels,
		SampleRate:    audio.SampleRate,
		ByteRate:      audio.SampleRate * audio.Channels * audio.BytesPerSample,
		BlockAlign:    audio.Channels * audio.BytesPerSample,
		BitsPerSample: bitsPerSample,
		Data:          [4]byte{'d', 'a', 't', 'a'},
		DataSize:      w.dataBytes,
	}

	if err := binary.Write(dst, binary.LittleEndian, header); err != nil {
		return fmt.Errorf("write wav header: %w", err)
	}
	return nil
}
