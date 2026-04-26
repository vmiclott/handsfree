// Package transcription provides speech-to-text functionality.
package transcription

// SpeechRecognizer recognizes speech from an audio stream.
type SpeechRecognizer interface {
	// Next returns the next transcription segment.
	// It returns io.EOF when speech detection is complete.
	Next() (segment string, err error)

	// Close stops the speech recognizer and releases resources.
	Close() error
}
