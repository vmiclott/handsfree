// Package audio provides audio processing functionality
package audio

// AudioSource is a streaming source of PCM audio samples.
//
// Implementations produce 16-bit little-endian mono audio frames.
// Callers must stop using the source after io.EOF.
type AudioSource interface {
	// Next returns the next chunk of audio samples.
	// It returns io.EOF when the stream is exhausted.
	Next() ([]int16, error)

	// Close stops the audio source and releases resources.
	Close() error
}
