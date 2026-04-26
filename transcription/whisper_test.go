package transcription

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

type fakeSpeechDetector struct {
	samples  [][]int16
	isSpeech []bool
	idx      int
	closed   bool
}

func (d *fakeSpeechDetector) Next() ([]int16, bool, error) {
	if d.idx >= len(d.samples) {
		return nil, false, io.EOF
	}
	samples := d.samples[d.idx]
	isSpeech := d.isSpeech[d.idx]
	d.idx++
	return samples, isSpeech, nil
}

func (d *fakeSpeechDetector) Close() error {
	d.closed = true
	return nil
}

func TestWhisperRecognizerConfigurationOptions(t *testing.T) {
	tests := []struct {
		name  string
		opt   WhisperConfigurationOption
		check func(*WhisperConfiguration) bool
	}{
		{
			name: "WithURL",
			opt:  WithURL("http://test:8090/transcribe"),
			check: func(c *WhisperConfiguration) bool {
				return c.URL == "http://test:8090/transcribe"
			},
		},
		{
			name: "WithSampleRate",
			opt:  WithSampleRate(48000),
			check: func(c *WhisperConfiguration) bool {
				return c.SampleRate == 48000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &WhisperConfiguration{
				URL:        "http://localhost:8080/transcribe",
				SampleRate: 16000,
			}
			tt.opt(config)
			if !tt.check(config) {
				t.Errorf("option %s did not set expected value", tt.name)
			}
		})
	}
}

func TestWhisperRecognizerClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte{'{', '"', 't', 'e', 'x', 't', '"', ':', '"', 'h', 'e', 'l', 'l', 'o', '"', '}'})
	}))
	defer server.Close()

	detector := &fakeSpeechDetector{
		samples: [][]int16{
			{100, 200, 300},
			{},
		},
		isSpeech: []bool{true, false},
	}

	recognizer, err := NewWhisperRecognizer(detector, WithURL(server.URL))
	if err != nil {
		t.Fatalf("NewWhisperRecognizer failed: %v", err)
	}

	for {
		_, err := recognizer.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
	}

	err = recognizer.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestWhisperRecognizerTranscription(t *testing.T) {
	responseBody := []byte{
		'{', '"', 's', 'e', 'g', 'm', 'e', 'n', 't', 's', '"', ':',
		'[', '{', '"', 't', 'e', 'x', 't', '"', ':', '"', 'h', 'e', 'l', 'l', 'o', ' ', 'w', 'o', 'r', 'l', 'd', '"', ',', '"', 's', 't', 'a', 'r', 't', '"', ':', '0', '.', '0', ',', '"', 'e', 'n', 'd', '"', ':', '1', '.', '0', '}', ']',
		',', '"', 'l', 'a', 'n', 'g', 'u', 'a', 'g', 'e', '"', ':', '"', 'e', 'n', '"', '}',
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(responseBody)
	}))
	defer server.Close()

	detector := &fakeSpeechDetector{
		samples: [][]int16{
			make([]int16, 16000),
			make([]int16, 16000),
		},
		isSpeech: []bool{true, false},
	}

	recognizer, err := NewWhisperRecognizer(detector, WithURL(server.URL))
	if err != nil {
		t.Fatalf("NewWhisperRecognizer failed: %v", err)
	}

	var texts []string
	for {
		text, err := recognizer.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Next error: %v", err)
		}
		if text != "" {
			texts = append(texts, text)
		}
	}

	if len(texts) != 1 {
		t.Fatalf("Expected 1 transcription, got %d", len(texts))
	}

	expected := "hello world"
	if texts[0] != expected {
		t.Errorf("Expected %q, got %q", expected, texts[0])
	}
}

func TestWhisperRecognizerReal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping real server test in short mode")
	}

	data, err := os.ReadFile("../testdata/hello_world_s16le_16kHz_mono.raw")
	if err != nil {
		t.Fatalf("failed to read audio file: %v", err)
	}

	samples := make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(data[i*2]) | int16(data[i*2+1])<<8
	}

	detector := &fakeSpeechDetector{
		samples: [][]int16{
			samples,
		},
		isSpeech: []bool{true},
	}

	recognizer, err := NewWhisperRecognizer(detector, WithURL("http://localhost:8080/transcribe"))
	if err != nil {
		t.Fatalf("NewWhisperRecognizer failed: %v", err)
	}
	defer func() { _ = recognizer.Close() }()

	var texts []string
	for {
		text, err := recognizer.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			// Skip on connection errors (server not running)
			if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "dial tcp") {
				t.Skip("whisper server not running on localhost:8080")
			}
			t.Fatalf("Next error: %v", err)
		}
		if text != "" {
			texts = append(texts, text)
		}
	}

	if len(texts) != 1 {
		t.Fatalf("Expected 1 transcription, got %d", len(texts))
	}

	expected := "Hello world."
	if texts[0] != expected {
		t.Errorf("Expected %q, got %q", expected, texts[0])
	}

	t.Logf("Transcribed: %v", texts)
}
