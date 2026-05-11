package main

import (
	"encoding/binary"
	"io"
)

type WEMHeader struct {
	Channels      int
	SampleRate    int
	TotalSamples  int
	PreSkip       int
	Packets       []OpusPacket
}

func writeWEM(w io.Writer, h WEMHeader) error {
	var dataSize uint32
	for _, p := range h.Packets {
		dataSize += uint32(len(p.Data))
	}

	seekSize := uint32(len(h.Packets) * 2)
	
	// Wwise Opus (0x3041) fmt chunk:
	// 0x00: wFormatTag (0x3041)
	// 0x02: nChannels
	// 0x04: nSamplesPerSec
	// 0x08: nAvgBytesPerSec
	// 0x0c: nBlockAlign
	// 0x0e: wBitsPerSample
	// 0x10: cbSize (extra data size)
	// 0x12: Extra Data:
	//   0x12: samples per frame (uint16)
	//   0x14: unknown (uint32)
	//   0x18: total samples (uint32)
	//   0x1c: table count (uint32)
	//   0x20: pre-skip (uint16)
	//   0x22: version (uint8, 1)
	//   0x23: mapping (uint8, 0 or mapping type)
	
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
	
	// Extra data (18 bytes)
	binary.Write(w, binary.LittleEndian, uint16(0)) // samples per frame (offset 0x12)
	binary.Write(w, binary.LittleEndian, uint32(0)) // unknown (offset 0x14)
	binary.Write(w, binary.LittleEndian, uint32(h.TotalSamples)) // offset 0x18
	binary.Write(w, binary.LittleEndian, uint32(len(h.Packets))) // offset 0x1c
	binary.Write(w, binary.LittleEndian, uint16(h.PreSkip))      // offset 0x20
	binary.Write(w, binary.LittleEndian, uint8(1))               // offset 0x22
	binary.Write(w, binary.LittleEndian, uint8(0))               // offset 0x23
	
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
