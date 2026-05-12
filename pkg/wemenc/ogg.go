package wemenc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type OpusPacket struct {
	Data    []byte
	Samples int
}

func parseOgg(r io.Reader) ([]OpusPacket, int, int, int, error) {
	var packets []OpusPacket
	var preSkip int
	var channels int
	var lastGranulePos int
	var currentPacket []byte

	for {
		header := make([]byte, 27)
		if _, err := io.ReadFull(r, header); err != nil {
			if err == io.EOF {
				break
			}
			return nil, 0, 0, 0, err
		}

		if string(header[0:4]) != "OggS" {
			return nil, 0, 0, 0, fmt.Errorf("invalid ogg signature")
		}

		granulePos := int64(binary.LittleEndian.Uint64(header[6:14]))
		// Only update lastGranulePos if it's a valid positive value
		if granulePos > 0 {
			lastGranulePos = int(granulePos)
		}

		segmentCount := int(header[26])
		lacingValues := make([]byte, segmentCount)
		if _, err := io.ReadFull(r, lacingValues); err != nil {
			return nil, 0, 0, 0, err
		}

		for _, l := range lacingValues {
			segment := make([]byte, int(l))
			if _, err := io.ReadFull(r, segment); err != nil {
				return nil, 0, 0, 0, err
			}
			currentPacket = append(currentPacket, segment...)
			
			// In Ogg, a packet ends if the segment length is < 255.
			if l < 255 {
				if len(currentPacket) > 0 {
					if bytes.HasPrefix(currentPacket, []byte("OpusHead")) {
						if len(currentPacket) >= 12 {
							preSkip = int(binary.LittleEndian.Uint16(currentPacket[10:12]))
							channels = int(currentPacket[9])
						}
					} else if bytes.HasPrefix(currentPacket, []byte("OpusTags")) {
						// ignore
					} else if len(currentPacket) > 8 { // Filter out very small bookkeeping/padding packets
						samples := getOpusSamples(currentPacket)
						packets = append(packets, OpusPacket{
							Data:    currentPacket,
							Samples: samples,
						})
					}
					currentPacket = nil
				}
			}
		}
	}

	return packets, preSkip, channels, lastGranulePos, nil
}

func getOpusSamples(data []byte) int {
	if len(data) < 1 {
		return 0
	}
	config := data[0] >> 3
	count := data[0] & 3
	var frames int
	if count == 0 {
		frames = 1
	} else if count <= 2 {
		frames = 2
	} else {
		if len(data) < 2 {
			return 0
		}
		frames = int(data[1] & 0x3F)
	}

	var samplesPerFrame int
	switch {
	case config < 12: // SILK
		ms := 0
		switch config & 3 {
		case 0: ms = 10
		case 1: ms = 20
		case 2: ms = 40
		case 3: ms = 60
		}
		samplesPerFrame = (48000 * ms) / 1000
	case config < 16: // Hybrid
		ms := 10
		if config&1 != 0 {
			ms = 20
		}
		samplesPerFrame = (48000 * ms) / 1000
	case config < 32: // CELT
		samplesPerFrame = (48000 << (config & 3)) / 400
	default:
		return 0
	}

	return samplesPerFrame * frames
}
