package wemenc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
)

// WEMHeader contains the metadata and audio data for a WEM file.
type WEMHeader struct {
	Channels     int
	SampleRate   int
	TotalSamples int
	PreSkip      int
	Packets      []OpusPacket
}

// EncodeOptions configures the encoding process.
type EncodeOptions struct {
	Bitrate string // e.g., "64k", "128k"
}

// EncodeToWEM encodes the audio from the input reader and writes the WEM data to the output writer.
// It uses ffmpeg for the Opus encoding step.
func EncodeToWEM(r io.Reader, w io.Writer, opt EncodeOptions) error {
	if opt.Bitrate == "" {
		opt.Bitrate = "64k"
	}

	// 1. Run FFmpeg to get Ogg Opus
	// We read from stdin and write Ogg to stdout
	cmd := exec.Command("ffmpeg", "-i", "pipe:0", "-c:a", libOpusOrOpus(), "-b:a", opt.Bitrate, "-application", "audio", "-f", "ogg", "pipe:1")
	cmd.Stdin = r
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr := &bytes.Buffer{}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return err
	}

	// 2. Parse Ogg stream
	packets, preSkip, channels, lastGranulePos, err := parseOgg(stdout)
	if err != nil {
		return fmt.Errorf("ogg parse error: %v (ffmpeg stderr: %s)", err, stderr.String())
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg error: %v (stderr: %s)", err, stderr.String())
	}

	// 3. Construct WEM structure
	header := WEMHeader{
		Channels:     channels,
		SampleRate:   48000,
		TotalSamples: lastGranulePos,
		PreSkip:      preSkip,
		Packets:      packets,
	}

	// 4. Write WEM to the provided writer
	return WriteWEM(w, header)
}

// WriteWEM serializes a WEMHeader into the Wwise RIFF format and writes it to w.
func WriteWEM(w io.Writer, h WEMHeader) error {
	var dataSize uint32
	for _, p := range h.Packets {
		dataSize += uint32(len(p.Data))
	}

	seekSize := uint32(len(h.Packets) * 2)
	fmtSize := uint32(36)
	cbSize := uint16(fmtSize - 18)
	riffSize := 4 + 8 + fmtSize + 8 + seekSize + 8 + dataSize

	// RIFF Header
	binary.Write(w, binary.BigEndian, []byte("RIFF"))
	binary.Write(w, binary.LittleEndian, uint32(riffSize))
	binary.Write(w, binary.BigEndian, []byte("WAVE"))

	// fmt chunk
	binary.Write(w, binary.BigEndian, []byte("fmt "))
	binary.Write(w, binary.LittleEndian, fmtSize)
	binary.Write(w, binary.LittleEndian, uint16(0x3041)) // OPUSWW
	binary.Write(w, binary.LittleEndian, uint16(h.Channels))
	binary.Write(w, binary.LittleEndian, uint32(h.SampleRate))
	binary.Write(w, binary.LittleEndian, uint32(0)) // AvgBytesPerSec
	binary.Write(w, binary.LittleEndian, uint16(0)) // BlockAlign
	binary.Write(w, binary.LittleEndian, uint16(0)) // BitsPerSample
	binary.Write(w, binary.LittleEndian, cbSize)

	// Extra data
	binary.Write(w, binary.LittleEndian, uint16(0))              // samples per frame
	binary.Write(w, binary.LittleEndian, uint32(0))              // unknown
	binary.Write(w, binary.LittleEndian, uint32(h.TotalSamples)) // total samples
	binary.Write(w, binary.LittleEndian, uint32(len(h.Packets))) // table count
	binary.Write(w, binary.LittleEndian, uint16(h.PreSkip))      // pre-skip
	binary.Write(w, binary.LittleEndian, uint8(1))               // version
	binary.Write(w, binary.LittleEndian, uint8(0))               // mapping

	// seek chunk
	binary.Write(w, binary.BigEndian, []byte("seek"))
	binary.Write(w, binary.LittleEndian, seekSize)
	for _, p := range h.Packets {
		binary.Write(w, binary.LittleEndian, uint16(len(p.Data)))
	}

	// data chunk
	binary.Write(w, binary.BigEndian, []byte("data"))
	binary.Write(w, binary.LittleEndian, dataSize)
	for _, p := range h.Packets {
		w.Write(p.Data)
	}

	return nil
}

func libOpusOrOpus() string {
	return "libopus"
}
