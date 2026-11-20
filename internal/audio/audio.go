// Package audio describes the audio format shared by the client and the server.
package audio

const (
	// SampleRate is the number of samples per second per channel.
	SampleRate = 48000

	// Channels is the number of audio channels; 1 means mono.
	Channels = 1

	// BytesPerSample is the size of one 16-bit PCM sample.
	BytesPerSample = 2

	// FrameMillis is the length of one Opus frame. Opus accepts only
	// 2.5, 5, 10, 20, 40 or 60 ms; 20 ms is the usual choice.
	FrameMillis = 20

	// FrameSamples is the number of samples in one Opus frame (960).
	FrameSamples = SampleRate * FrameMillis / 1000 * Channels

	// FrameBytes is the size of one Opus frame as raw PCM (1920 bytes).
	FrameBytes = FrameSamples * BytesPerSample

	// MaxPacketBytes is an upper bound for one encoded Opus packet.
	MaxPacketBytes = 4000
)
