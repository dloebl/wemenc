package wemenc

import (
	"bytes"
	"testing"
)

func TestEncodeToWEM_UnsupportedCodec(t *testing.T) {
	r := bytes.NewReader([]byte("mock audio data"))
	var w bytes.Buffer

	unsupportedCodecs := []Codec{CodecPCM, CodecADPCM, CodecVorbis}

	for _, codec := range unsupportedCodecs {
		opt := EncodeOptions{
			Codec: codec,
		}
		err := EncodeToWEM(r, &w, opt)
		if err == nil {
			t.Errorf("expected error for codec %v, but got nil", codec)
		}
	}
}

func TestCodecConstants(t *testing.T) {
	if CodecPCM != 0x0001 {
		t.Errorf("CodecPCM: expected 0x0001, got 0x%04X", CodecPCM)
	}
	if CodecADPCM != 0x0002 {
		t.Errorf("CodecADPCM: expected 0x0002, got 0x%04X", CodecADPCM)
	}
	if CodecVorbis != 0xFFFF {
		t.Errorf("CodecVorbis: expected 0xFFFF, got 0x%04X", CodecVorbis)
	}
	if CodecOpus != 0x3041 {
		t.Errorf("CodecOpus: expected 0x3041, got 0x%04X", CodecOpus)
	}
}
