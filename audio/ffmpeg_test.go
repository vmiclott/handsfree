package audio

import (
	"io"
	"os"
	"testing"
)

const (
	wavAudioFile = "../testdata/hello_world.wav"
	rawAudioFile = "../testdata/hello_world_s16le_16kHz_mono.raw"
)

func TestFFmpegAudioSourceConfigurationOptions(t *testing.T) {
	tests := []struct {
		name  string
		opt   FFmpegAudioSourceConfigurationOption
		check func(*FFmpegAudioSourceConfiguration) bool
	}{
		{
			name: "WithInputFormat",
			opt:  WithInputFormat("pulse"),
			check: func(c *FFmpegAudioSourceConfiguration) bool {
				return c.InputFormat == "pulse"
			},
		},
		{
			name: "WithInputSource",
			opt:  WithInputSource("hw:0"),
			check: func(c *FFmpegAudioSourceConfiguration) bool {
				return c.InputSource == "hw:0"
			},
		},
		{
			name: "WithChunkSize",
			opt:  WithChunkSize(6400),
			check: func(c *FFmpegAudioSourceConfiguration) bool {
				return c.ChunkSize == 6400
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &FFmpegAudioSourceConfiguration{
				InputFormat: "alsa",
				InputSource: "default",
				ChunkSize:   3200,
			}
			tt.opt(config)
			if !tt.check(config) {
				t.Errorf("option %s did not set expected value", tt.name)
			}
		})
	}
}

func TestFFmpegAudioSourceClose(t *testing.T) {
	source, err := NewFFmpegAudioSource(
		WithInputFormat("wav"),
		WithInputSource(wavAudioFile),
	)
	if err != nil {
		t.Fatalf("NewFFmpegAudioSource failed: %v", err)
	}

	for {
		_, err := source.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
	}

	err = source.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestWavAndRawProduceSameAudio(t *testing.T) {
	chunkSize := 3200
	rawFileData, err := os.ReadFile(rawAudioFile)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}

	rawSamples := make([]int16, len(rawFileData)/2)
	for i := range rawSamples {
		rawSamples[i] = int16(rawFileData[i*2]) | int16(rawFileData[i*2+1])<<8
	}

	wavSource, err := NewFFmpegAudioSource(
		WithInputFormat("wav"),
		WithInputSource(wavAudioFile),
		WithChunkSize(chunkSize),
	)
	if err != nil {
		t.Fatalf("NewFFmpegAudioSource failed: %v", err)
	}
	defer func() {
		if err := wavSource.Close(); err != nil {
			t.Logf("Close error: %v", err)
		}
	}()

	var wavSamples []int16
	for {
		samples, err := wavSource.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next error: %v", err)
		}
		wavSamples = append(wavSamples, samples...)
	}

	if len(wavSamples) > len(rawSamples) {
		t.Fatalf("wav produced more samples than raw file")
	}

	for i := range wavSamples {
		if wavSamples[i] != rawSamples[i] {
			t.Errorf("sample mismatch at index %d: wav=%d raw=%d", i, wavSamples[i], rawSamples[i])
			return
		}
	}
}
