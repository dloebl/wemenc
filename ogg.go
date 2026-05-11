package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type OggPage struct {
	HeaderType    byte
	GranulePos    int64
	Serial        uint32
	Sequence      uint32
	Checksum      uint32
	Segments      []byte
	Data          []byte
}

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

		// headerType := header[5]
		granulePos := int64(binary.LittleEndian.Uint64(header[6:14]))
		lastGranulePos = int(granulePos)
		// serial := binary.LittleEndian.Uint32(header[14:18])
		// sequence := binary.LittleEndian.Uint32(header[18:22])
		// checksum := binary.LittleEndian.Uint32(header[22:26])
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
			if l < 255 {
				if len(currentPacket) > 0 {
					if bytes.HasPrefix(currentPacket, []byte("OpusHead")) {
						if len(currentPacket) >= 12 {
							preSkip = int(binary.LittleEndian.Uint16(currentPacket[10:12]))
							channels = int(currentPacket[9])
						}
					} else if bytes.HasPrefix(currentPacket, []byte("OpusTags")) {
						// ignore
					} else {
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

// Simplified version of opus_packet_get_samples_per_frame from vgmstream
func getOpusSamples(data []byte) int {
	if len(data) < 1 {
		return 0
	}
	
	fs := 48000
	var audiosize int
	if data[0]&0x80 != 0 {
		audiosize = int((data[0] >> 3) & 0x3)
		audiosize = (fs << audiosize) / 400
	} else if (data[0] & 0x60) == 0x60 {
		if data[0]&0x08 != 0 {
			audiosize = fs / 50
		} else {
			audiosize = fs / 100
		}
	} else {
		audiosize = int((data[0] >> 3) & 0x3)
		if audiosize == 3 {
			audiosize = fs * 60 / 1000
		} else {
			audiosize = (fs << audiosize) / 100
		}
	}

	count := int(data[0] & 0x3)
	var nbframes int
	if count == 0 {
		nbframes = 1
	} else if count != 3 {
		nbframes = 2
	} else if len(data) < 2 {
		nbframes = 0
	} else {
		nbframes = int(data[1] & 0x3F)
	}

	return audiosize * nbframes
}
