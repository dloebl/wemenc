package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

func encodeToWEM(inputPath, outputPath string, bitrate string) error {
	// 1. Run FFmpeg to get Ogg Opus
	cmd := exec.Command("ffmpeg", "-i", inputPath, "-c:a", libOpusOrOpus(), "-b:a", bitrate, "-application", "audio", "-f", "ogg", "-")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	// 2. Parse Ogg stream
	packets, preSkip, channels, lastGranulePos, err := parseOgg(stdout)
	if err != nil {
		// Read stderr for better error message
		slurp, _ := io.ReadAll(stderr)
		return fmt.Errorf("ogg parse error: %v (ffmpeg stderr: %s)", err, string(slurp))
	}

	if err := cmd.Wait(); err != nil {
		slurp, _ := io.ReadAll(stderr)
		return fmt.Errorf("ffmpeg error: %v (stderr: %s)", err, string(slurp))
	}

	// 3. Write WEM
	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	header := WEMHeader{
		Channels:     channels,
		SampleRate:   48000,
		TotalSamples: lastGranulePos,
		PreSkip:      preSkip,
		Packets:      packets,
	}
	
	return writeWEM(outFile, header)
}

func libOpusOrOpus() string {
	// Check if libopus is available, otherwise use opus
	// For most ffmpeg builds, it's libopus.
	return "libopus"
}

// In a real app, we'd use ffprobe. For this task, we can try to extract it from OpusHead in parseOgg.
// Let's modify parseOgg to return channel count.
func getChannelsFromOpusHead(data []byte) int {
	if len(data) >= 10 && string(data[0:8]) == "OpusHead" {
		return int(data[9])
	}
	return 2
}
