package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dloebl/wemenc/pkg/wemenc"
)

func main() {
	inputFlag := flag.String("i", "", "Input audio file")
	outputFlag := flag.String("o", "", "Output WEM file")
	bitrateFlag := flag.String("b", "64k", "Bitrate (e.g. 64k, 128k)")
	codecFlag := flag.String("c", "opus", "Codec (opus, pcm, adpcm, vorbis)")

	flag.Parse()

	if *inputFlag == "" || *outputFlag == "" {
		fmt.Println("Usage: wemenc -i <input> -o <output> [-b <bitrate>] [-c <codec>]")
		os.Exit(1)
	}

	var codec wemenc.Codec
	switch *codecFlag {
	case "opus":
		codec = wemenc.CodecOpus
	case "pcm":
		codec = wemenc.CodecPCM
	case "adpcm":
		codec = wemenc.CodecADPCM
	case "vorbis":
		codec = wemenc.CodecVorbis
	default:
		fmt.Fprintf(os.Stderr, "Unknown codec: %s\n", *codecFlag)
		os.Exit(1)
	}

	fmt.Printf("Encoding %s to %s (codec: %s, bitrate: %s)...\n", *inputFlag, *outputFlag, *codecFlag, *bitrateFlag)

	inFile, err := os.Open(*inputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input: %v\n", err)
		os.Exit(1)
	}
	defer inFile.Close()

	outFile, err := os.Create(*outputFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	opt := wemenc.EncodeOptions{
		Codec:   codec,
		Bitrate: *bitrateFlag,
	}

	if err := wemenc.EncodeToWEM(inFile, outFile, opt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Done!")
}
