// Package vad provides voice activity detection functionality.
package vad

// SpeechDetector detects voice activity in audio samples from an AudioSource.
type SpeechDetector interface {
	// Next returns the next chunk of audio samples.
	// The isSpeech return value indicates whether the chunk contains speech.
	// It returns io.EOF when the stream is exhausted.
	Next() (samples []int16, isSpeech bool, err error)

	// Close stops the speech detector and releases resources.
	Close() error
}
