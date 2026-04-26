// Package vad provides voice activity detection functionality.
package vad

import (
	"log/slog"
	"math"

	"github.com/vmiclott/handsfree/audio"
)

// RMSEnergyDetectorConfiguration defines runtime configuration for an RMSEnergyDetector.
type RMSEnergyDetectorConfiguration struct {
	Threshold float64
}

// RMSEnergyDetectorConfigurationOption is a functional option used to configure
// an RMSEnergyDetectorConfiguration in a fluent and composable way.
type RMSEnergyDetectorConfigurationOption func(*RMSEnergyDetectorConfiguration)

// WithThreshold sets the RMS energy threshold above which audio is considered speech.
// A threshold of 500 is typical for voice detection.
func WithThreshold(threshold float64) RMSEnergyDetectorConfigurationOption {
	return func(config *RMSEnergyDetectorConfiguration) {
		config.Threshold = threshold
	}
}

// RMSEnergyDetector is a SpeechDetector implementation that uses RMS energy levels to detect speech.
type RMSEnergyDetector struct {
	source    audio.AudioSource
	threshold float64
}

// NewRMSEnergyDetector creates a new RMS energy-based speech detector that wraps an AudioSource.
//
// Optional configuration can be provided via functional options.
// A default threshold of 500 is used if not specified.
//
// The caller must call Close to release underlying resources.
func NewRMSEnergyDetector(source audio.AudioSource, opts ...RMSEnergyDetectorConfigurationOption) (SpeechDetector, error) {
	config := RMSEnergyDetectorConfiguration{
		Threshold: 500,
	}
	for _, opt := range opts {
		opt(&config)
	}

	slog.Info("RMSEnergyDetector created", "threshold", config.Threshold)

	return &RMSEnergyDetector{
		source:    source,
		threshold: config.Threshold,
	}, nil
}

// Next returns the next chunk of audio samples with speech detection.
func (d *RMSEnergyDetector) Next() ([]int16, bool, error) {
	samples, err := d.source.Next()
	if err != nil {
		return nil, false, err
	}

	rms := computeRMS(samples)
	isSpeech := rms > d.threshold

	slog.Debug("VAD processed", "samples", len(samples), "rms", rms, "threshold", d.threshold, "isSpeech", isSpeech)

	return samples, isSpeech, nil
}

// Close stops the audio source and releases resources.
func (d *RMSEnergyDetector) Close() error {
	slog.Info("RMSEnergyDetector closing")
	return d.source.Close()
}

// computeRMS computes the RMS energy of int16 samples.
func computeRMS(samples []int16) float64 {
	if len(samples) == 0 {
		return 0
	}

	var sum float64
	for _, s := range samples {
		sum += float64(s) * float64(s)
	}
	return math.Sqrt(sum / float64(len(samples)))
}
