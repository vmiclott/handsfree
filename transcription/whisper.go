// Package transcription provides speech-to-text functionality.
package transcription

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/vmiclott/handsfree/vad"
)

// WhisperSegment represents a transcribed segment of audio for the Whisper model.
type WhisperSegment struct {
	Text  string  `json:"text"`
	Start float64 `json:"start"`
	End   float64 `json:"end"`
}

// WhisperResponse represents the full response from a Whisper transcription server.
type WhisperResponse struct {
	Segments []WhisperSegment `json:"segments"`
	Info     any              `json:"info"`
}

// WhisperConfiguration defines runtime configuration for a Whisper recognizer.
type WhisperConfiguration struct {
	URL        string
	SampleRate int
}

// WhisperConfigurationOption is a functional option used to configure
// a WhisperConfiguration in a fluent and composable way.
type WhisperConfigurationOption func(*WhisperConfiguration)

// WithURL sets the Whisper server URL.
func WithURL(url string) WhisperConfigurationOption {
	return func(config *WhisperConfiguration) {
		config.URL = url
	}
}

// WithSampleRate sets the audio sample rate.
func WithSampleRate(sampleRate int) WhisperConfigurationOption {
	return func(config *WhisperConfiguration) {
		config.SampleRate = sampleRate
	}
}

// WhisperRecognizer is a SpeechRecognizer implementation that uses OpenAI Whisper.
type WhisperRecognizer struct {
	url        string
	sampleRate int
	detector   vad.SpeechDetector
	speechBuf  []int16
}

// NewWhisperRecognizer creates a new Whisper-based speech recognizer.
//
// Optional configuration can be provided via functional options.
// A default URL of "localhost:8080/transcribe" is used if not specified.
//
// The caller must call Close to release underlying resources.
func NewWhisperRecognizer(detector vad.SpeechDetector, opts ...WhisperConfigurationOption) (SpeechRecognizer, error) {
	config := WhisperConfiguration{
		URL:        "http://localhost:8080/transcribe",
		SampleRate: 16000,
	}
	for _, opt := range opts {
		opt(&config)
	}

	slog.Info("WhisperRecognizer created", "url", config.URL, "sampleRate", config.SampleRate)

	return &WhisperRecognizer{
		url:        config.URL,
		sampleRate: config.SampleRate,
		detector:   detector,
	}, nil
}

// Next processes the next chunk of audio and returns a transcription if speech is detected.
func (recognizer *WhisperRecognizer) Next() (string, error) {
	samples, isSpeech, err := recognizer.detector.Next()
	if err != nil {
		if err == io.EOF && len(recognizer.speechBuf) > 0 {
			return recognizer.transcribe()
		}
		return "", err
	}

	if isSpeech {
		recognizer.speechBuf = append(recognizer.speechBuf, samples...)
		return "", nil
	}

	if len(recognizer.speechBuf) == 0 {
		return "", nil
	}

	chunkDurationMs := len(samples) * 1000 / recognizer.sampleRate
	silenceMs := 1000

	if chunkDurationMs < silenceMs {
		return "", nil
	}

	return recognizer.transcribe()
}

func (recognizer *WhisperRecognizer) transcribe() (string, error) {
	if len(recognizer.speechBuf) == 0 {
		return "", nil
	}

	slog.Info("transcribing segment", "samples", len(recognizer.speechBuf), "durationMs", len(recognizer.speechBuf)*1000/recognizer.sampleRate)

	audioBytes := int16ToBytes(recognizer.speechBuf)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s?sample_rate=%d", recognizer.url, recognizer.sampleRate)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(audioBytes))
	if err != nil {
		slog.Error("failed to create request", "error", err)
		recognizer.speechBuf = nil
		return "", err
	}
	req.Header.Set("Content-Type", "application/octet-stream")

	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Error("transcription request failed", "error", err)
		recognizer.speechBuf = nil
		return "", err
	}
	defer func() { _ = httpResp.Body.Close() }()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		slog.Error("transcription failed", "status", httpResp.StatusCode, "body", string(body))
		recognizer.speechBuf = nil
		return "", fmt.Errorf("server returned %d", httpResp.StatusCode)
	}

	var result WhisperResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		slog.Error("failed to decode response", "error", err)
		recognizer.speechBuf = nil
		return "", err
	}

	var text string
	for _, seg := range result.Segments {
		text += seg.Text + " "
	}
	text = strings.TrimSpace(text)

	slog.Info("transcribed", "text", text, "segments", len(result.Segments))

	recognizer.speechBuf = nil
	return text, nil
}

// Close stops the speech recognizer and releases resources.
func (recognizer *WhisperRecognizer) Close() error {
	if len(recognizer.speechBuf) > 0 {
		_, _ = recognizer.transcribe()
	}
	slog.Info("WhisperRecognizer closing")

	if recognizer.detector != nil {
		return recognizer.detector.Close()
	}
	return nil
}

// int16ToBytes converts int16 samples to little-endian bytes.
func int16ToBytes(samples []int16) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, samples)
	return buf.Bytes()
}
