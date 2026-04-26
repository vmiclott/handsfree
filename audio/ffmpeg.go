package audio

import (
	"io"
	"log/slog"
	"os"
	"os/exec"
	"time"
)

// FFmpegAudioSourceConfiguration defines runtime configuration for an FFmpeg-backed audio source.
//
// It controls how audio is captured and how it is framed into PCM chunks.
//
// InputFormat specifies the FFmpeg input device or demuxer (e.g. "alsa", "pulse", "wav").
// InputSource specifies the device name or file path.
// ChunkSize defines the number of bytes read per frame (must be even for int16 decoding).
type FFmpegAudioSourceConfiguration struct {
	InputFormat string
	InputSource string
	ChunkSize   int
}

// FFmpegAudioSourceConfigurationOption is a functional option used to configure
// an FFmpegAudioSourceConfiguration in a fluent and composable way.
type FFmpegAudioSourceConfigurationOption func(*FFmpegAudioSourceConfiguration)

// WithInputFormat sets the FFmpeg input format (e.g. "alsa", "pulse", "wav").
func WithInputFormat(inputFormat string) FFmpegAudioSourceConfigurationOption {
	return func(config *FFmpegAudioSourceConfiguration) {
		config.InputFormat = inputFormat
	}
}

// WithInputSource sets the FFmpeg input source (device name or file path).
func WithInputSource(inputSource string) FFmpegAudioSourceConfigurationOption {
	return func(config *FFmpegAudioSourceConfiguration) {
		config.InputSource = inputSource
	}
}

// WithChunkSize sets the maximum number of bytes read per audio frame.
// Each chunk will be decoded into int16 PCM samples.
func WithChunkSize(chunkSize int) FFmpegAudioSourceConfigurationOption {
	return func(config *FFmpegAudioSourceConfiguration) {
		config.ChunkSize = chunkSize
	}
}

// FFmpegAudioSource is an AudioSource implementation backed by an FFmpeg subprocess.
//
// It reads raw PCM audio (s16le, mono, 16kHz) from FFmpeg's stdout and
// exposes it as fixed-size chunks of int16 samples.
//
// The underlying FFmpeg process is started at creation time and must be
// explicitly stopped using Close.
type FFmpegAudioSource struct {
	cmd       *exec.Cmd
	stdout    io.ReadCloser
	buf       []byte
	chunkSize int
}

// NewFFmpegAudioSource creates and starts a new FFmpeg-based audio source.
//
// It launches an FFmpeg subprocess configured to output raw PCM audio
// (s16le, mono, 16kHz) to a pipe, which is then consumed in fixed-size chunks.
//
// Optional configuration can be provided via functional options.
//
// The caller must call Close to release underlying resources.
func NewFFmpegAudioSource(opts ...FFmpegAudioSourceConfigurationOption) (*FFmpegAudioSource, error) {
	config := FFmpegAudioSourceConfiguration{
		InputFormat: "alsa",
		InputSource: "default",
		ChunkSize:   3200, // 100ms of single-channel s16le
	}
	for _, opt := range opts {
		opt(&config)
	}
	cmd := buildCommand(config)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("failed to create stdout pipe", "error", err)
		return nil, err
	}

	err = cmd.Start()
	if err != nil {
		slog.Error("failed to start ffmpeg", "error", err)
		return nil, err
	}

	slog.Info("ffmpeg started", "inputFormat", config.InputFormat, "inputSource", config.InputSource, "chunkSize", config.ChunkSize)

	return &FFmpegAudioSource{
		cmd:       cmd,
		stdout:    stdout,
		buf:       make([]byte, config.ChunkSize),
		chunkSize: config.ChunkSize,
	}, nil
}

// Next reads the next chunk of PCM audio samples from the FFmpeg process.
//
// Each call returns a slice of int16 samples decoded from little-endian PCM data.
//
// Return behavior:
//   - ([]int16, nil): normal audio frame
//   - (nil, io.EOF): no more data is available
//   - (nil, error): underlying read or process error
//
// The returned slice is only valid until the next call to Next.
func (source *FFmpegAudioSource) Next() ([]int16, error) {
	buf := source.buf[:source.chunkSize]

	n, err := io.ReadFull(source.stdout, buf)
	if err == io.ErrUnexpectedEOF {
		if n == 0 {
			slog.Info("ffmpeg stream completed")
			return nil, io.EOF
		}
		buf = buf[:n]
	} else if err != nil {
		slog.Error("ffmpeg read error", "error", err)
		return nil, err
	}

	samples := make([]int16, len(buf)/2)

	for i := range samples {
		j := i * 2
		samples[i] = int16(buf[j]) | int16(buf[j+1])<<8
	}
	return samples, nil
}

// Close stops the FFmpeg process and releases all associated resources.
//
// It first sends a SIGINT to allow graceful shutdown. If FFmpeg does not exit
// within 1 second, it is forcefully killed.
//
// Close waits for the process to exit before returning.
//
// It is safe to call Close multiple times; subsequent calls are no-ops.
func (source *FFmpegAudioSource) Close() error {
	if source.cmd == nil || source.cmd.Process == nil {
		return nil
	}
	_ = source.cmd.Process.Signal(os.Interrupt)
	_ = source.stdout.Close()
	done := make(chan error, 1)
	go func() {
		done <- source.cmd.Wait()
	}()

	select {
	case err := <-done:
		slog.Info("ffmpeg stopped", "error", err)
		return err
	case <-time.After(time.Second):
		slog.Warn("ffmpeg did not stop gracefully, killing")
		_ = source.cmd.Process.Kill()
		return source.cmd.Wait()
	}
}

func buildCommand(config FFmpegAudioSourceConfiguration) *exec.Cmd {
	return exec.Command(
		"ffmpeg",
		"-f", config.InputFormat,
		"-i", config.InputSource,
		"-f", "s16le",
		"-ac", "1",
		"-ar", "16000",
		"pipe:1",
	)
}
