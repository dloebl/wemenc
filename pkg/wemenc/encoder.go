package wemenc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// Codec represents the audio codec used in the WEM file.
type Codec uint16

const (
	CodecPCM    Codec = 0x0001
	CodecADPCM  Codec = 0x0002
	CodecVorbis Codec = 0xFFFF
	CodecOpus   Codec = 0x3041
)

// WEMHeader contains the metadata and audio data for a WEM file.
type WEMHeader struct {
	Codec           Codec
	Channels        int
	SampleRate      int
	AvgBytesPerSec  uint32
	TotalSamples    int
	PreSkip         int
	SamplesPerFrame uint16
	Packets         []OpusPacket
}

// EncodeOptions configures the encoding process.
type EncodeOptions struct {
	Codec      Codec
	Bitrate    string // e.g., "64k", "128k"
	FFmpegPath string // Path to ffmpeg binary, if empty uses "ffmpeg" from PATH
}

// EncodeToWEM encodes the audio from the input reader and writes the WEM data to the output writer.
// It uses ffmpeg for the Opus encoding step.
func EncodeToWEM(r io.Reader, w io.Writer, opt EncodeOptions) error {
	if opt.Codec == 0 {
		opt.Codec = CodecOpus
	}

	if opt.Codec != CodecOpus {
		return fmt.Errorf("codec %v is not supported yet", opt.Codec)
	}

	if opt.Bitrate == "" {
		opt.Bitrate = "96k"
	}

	ffmpeg := opt.FFmpegPath
	if ffmpeg == "" {
		ffmpeg = "ffmpeg"
	}

	// Calculate bitrate for header
	var bitrate uint32 = 96000
	fmt.Sscanf(opt.Bitrate, "%dk", &bitrate)
	if strings.HasSuffix(opt.Bitrate, "k") {
		bitrate *= 1000
	}

	// 1. Run FFmpeg to get Ogg Opus
	// We force mono, 48kHz, CBR, and 20ms frames for maximum compatibility.
	cmd := exec.Command(ffmpeg, "-i", "pipe:0", "-ac", "1", "-ar", "48000", "-c:a", libOpusOrOpus(), "-b:a", opt.Bitrate, "-vbr", "off", "-application", "audio", "-frame_duration", "20", "-f", "ogg", "pipe:1")
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
	packets, preSkip, channels, _, err := parseOgg(stdout)
	if err != nil {
		return fmt.Errorf("ogg parse error: %v (ffmpeg stderr: %s)", err, stderr.String())
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg error: %v (stderr: %s)", err, stderr.String())
	}

	// Calculate playable samples accurately
	// Wwise-Opus expects (PacketCount * 960) - PreSkip
	totalSamples := (len(packets) * 960) - preSkip
	if totalSamples < 0 {
		totalSamples = 0
	}

	// 3. Construct WEM structure
	header := WEMHeader{
		Codec:           opt.Codec,
		Channels:        channels,
		SampleRate:      48000,
		AvgBytesPerSec:  bitrate / 8,
		TotalSamples:    totalSamples,
		PreSkip:         preSkip,
		SamplesPerFrame: 960,
		Packets:         packets,
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
	hashSize := uint32(16)
	cbSize := uint16(fmtSize - 18)

	// Helper to calculate padding needed to align to 4 bytes
	padding := func(size uint32) uint32 {
		if size%4 == 0 {
			return 0
		}
		return 4 - (size % 4)
	}

	fmtPadding := padding(fmtSize)
	hashPadding := padding(hashSize)
	seekPadding := padding(seekSize)

	// RIFF size is total file size - 8 bytes
	riffSize := 4 +
		(8 + fmtSize + fmtPadding) +
		(8 + hashSize + hashPadding) +
		(8 + seekSize + seekPadding) +
		(8 + dataSize)

	// RIFF Header
	binary.Write(w, binary.BigEndian, []byte("RIFF"))
	binary.Write(w, binary.LittleEndian, uint32(riffSize))
	binary.Write(w, binary.BigEndian, []byte("WAVE"))

	// fmt chunk
	binary.Write(w, binary.BigEndian, []byte("fmt "))
	binary.Write(w, binary.LittleEndian, fmtSize)
	binary.Write(w, binary.LittleEndian, uint16(h.Codec))
	binary.Write(w, binary.LittleEndian, uint16(h.Channels))
	binary.Write(w, binary.LittleEndian, uint32(h.SampleRate))
	binary.Write(w, binary.LittleEndian, uint32(h.AvgBytesPerSec))
	binary.Write(w, binary.LittleEndian, uint16(0)) // BlockAlign
	binary.Write(w, binary.LittleEndian, uint16(0)) // BitsPerSample
	binary.Write(w, binary.LittleEndian, cbSize)

	// Extra data
	binary.Write(w, binary.LittleEndian, uint16(h.SamplesPerFrame)) // samples per frame
	binary.Write(w, binary.LittleEndian, uint32(0x00004101))       // unknown (Opus constant)
	binary.Write(w, binary.LittleEndian, uint32(h.TotalSamples))    // total samples
	binary.Write(w, binary.LittleEndian, uint32(len(h.Packets)))    // table count
	binary.Write(w, binary.LittleEndian, uint16(h.PreSkip))         // pre-skip
	binary.Write(w, binary.LittleEndian, uint8(1))                  // version
	binary.Write(w, binary.LittleEndian, uint8(0))                  // mapping (0 for mono/stereo)
	if fmtPadding > 0 {
		w.Write(make([]byte, fmtPadding))
	}

	// hash chunk
	binary.Write(w, binary.BigEndian, []byte("hash"))
	binary.Write(w, binary.LittleEndian, hashSize)
	binary.Write(w, binary.LittleEndian, [16]byte{}) // dummy hash
	if hashPadding > 0 {
		w.Write(make([]byte, hashPadding))
	}

	// seek chunk
	binary.Write(w, binary.BigEndian, []byte("seek"))
	binary.Write(w, binary.LittleEndian, seekSize)
	for _, p := range h.Packets {
		binary.Write(w, binary.LittleEndian, uint16(len(p.Data)))
	}
	if seekPadding > 0 {
		w.Write(make([]byte, seekPadding))
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
