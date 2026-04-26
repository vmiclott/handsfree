package vad

import (
	"io"
	"testing"

	"github.com/vmiclott/handsfree/audio"
)

func TestRMSEnergyDetectorConfigurationOptions(t *testing.T) {
	tests := []struct {
		name  string
		opt   RMSEnergyDetectorConfigurationOption
		check func(*RMSEnergyDetectorConfiguration) bool
	}{
		{
			name: "WithThreshold",
			opt:  WithThreshold(1000),
			check: func(c *RMSEnergyDetectorConfiguration) bool {
				return c.Threshold == 1000
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &RMSEnergyDetectorConfiguration{
				Threshold: 500,
			}
			tt.opt(config)
			if !tt.check(config) {
				t.Errorf("option %s did not set expected value", tt.name)
			}
		})
	}
}

func TestRMSEnergyDetectorClose(t *testing.T) {
	source, err := audio.NewFFmpegAudioSource(
		audio.WithInputFormat("wav"),
		audio.WithInputSource("../testdata/hello_world.wav"),
	)
	if err != nil {
		t.Fatalf("NewFFmpegAudioSource failed: %v", err)
	}

	detector, err := NewRMSEnergyDetector(source)
	if err != nil {
		t.Fatalf("NewRMSEnergyDetector failed: %v", err)
	}

	for {
		_, _, err := detector.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
	}

	err = detector.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestRMSEnergyDetectorSpeechDetection(t *testing.T) {
	source, err := audio.NewFFmpegAudioSource(
		audio.WithInputFormat("wav"),
		audio.WithInputSource("../testdata/hello_world.wav"),
		audio.WithChunkSize(3200),
	)
	if err != nil {
		t.Fatalf("NewFFmpegAudioSource failed: %v", err)
	}
	defer func() {
		if err := source.Close(); err != nil {
			t.Logf("Close error: %v", err)
		}
	}()

	detector, err := NewRMSEnergyDetector(source, WithThreshold(100))
	if err != nil {
		t.Fatalf("NewRMSEnergyDetector failed: %v", err)
	}
	defer func() {
		if err := detector.Close(); err != nil {
			t.Logf("Close error: %v", err)
		}
	}()

	expected := []bool{false, true, true, true, true, true, true, true, false}

	var results []bool
	for {
		_, isSpeech, err := detector.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		results = append(results, isSpeech)
	}

	if len(results) != len(expected) {
		t.Fatalf("Expected %d chunks, got %d", len(expected), len(results))
	}

	for i, want := range expected {
		got := results[i]
		if got != want {
			t.Errorf("Chunk %d: isSpeech=%v, want %v", i, got, want)
		}
	}
}
