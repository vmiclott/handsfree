# HandsFree

Hands-free voice-controlled typing and keyboard command execution.

## Development

Use `make` to format, vet, lint, test and build the application.

| Target  | Description                                                                                          |
| ------- | ---------------------------------------------------------------------------------------------------- |
| `all`   | Run `fmt`, `vet`, `lint`, `test` and `build` targets                                                 |
| `fmt`   | Format the code with `go fmt`                                                                        |
| `vet`   | Examine the code `go vet` for suspicious constructs and potential bugs                               |
| `lint`  | Lint the code with `golangci-lint`                                                                   |
| `test`  | Test the code with `go test` and write coverage report to `build/coverage.html` with `go tool cover` |
| `build` | Build the application to `build/handsfree`                                                           |

## VAD

Uses RMS energy detection. Speech is detected when RMS > threshold (default: 500 for int16 PCM).

## Transcription

Buffers speech segments and sends to [whisper-server](https://github.com/vmiclott/whisper-server) for transcription.
