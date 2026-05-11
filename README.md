# wemenc

`wemenc` is a command-line tool written in Go for encoding audio files into the Wwise WEM format (Wwise Encoded Media). Currently, **only the Opus codec is supported**. It leverages `ffmpeg` for the underlying Opus encoding and packages the result into a Wwise-compatible RIFF container.

## Features

- Encodes any audio format supported by `ffmpeg` to WEM Opus.
- Automatically handles Ogg/Opus stream parsing to extract raw packets.
- Generates required Wwise chunks: `fmt`, `seek` (packet table), and `data`.
- Precise sample count handling using Ogg granule positions.

## Prerequisites

- **Go**: 1.25 or later.
- **FFmpeg**: Must be installed and available in your `PATH`. The tool expects `libopus` support in FFmpeg.

## Installation

Clone the repository and build the binary:

```bash
go build -o wemenc ./cmd/wemenc
```

## Usage

```bash
./wemenc -i <input_file> -o <output_file.wem> [-b <bitrate>]
```

### Options

- `-i`: Path to the input audio file (e.g., `.wav`, `.mp3`).
- `-o`: Path to the output WEM file.
- `-b`: Bitrate for the Opus encoder (default: `64k`). You can use values like `96k`, `128k`, etc.

### Example

```bash
./wemenc -i music.wav -o music.wem -b 128k
```

## How it works

1.  **FFmpeg Orchestration**: The tool runs `ffmpeg` as a subprocess, pipe-encoding the input audio to an Ogg Opus stream.
2.  **Ogg Demuxing**: A custom parser reads the Ogg stream from FFmpeg's stdout, extracting individual Opus packets and metadata like `pre-skip` and channel count.
3.  **WEM Packaging**: The extracted packets are wrapped into a RIFF/WAVE container with Wwise-specific headers and a packet size table (`seek` chunk), making it compatible with games using the Audiokinetic Wwise engine and tools like `vgmstream`.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
