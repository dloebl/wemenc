package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	inputFlag := flag.String("i", "", "Input audio file")
	outputFlag := flag.String("o", "", "Output WEM file")
	bitrateFlag := flag.String("b", "64k", "Bitrate (e.g. 64k, 128k)")
	
	flag.Parse()
	
	if *inputFlag == "" || *outputFlag == "" {
		fmt.Println("Usage: wemenc -i <input> -o <output> [-b <bitrate>]")
		os.Exit(1)
	}
	
	fmt.Printf("Encoding %s to %s (bitrate: %s)...\n", *inputFlag, *outputFlag, *bitrateFlag)
	
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

	opt := EncodeOptions{
		Bitrate: *bitrateFlag,
	}

	if err := EncodeToWEM(inFile, outFile, opt); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Done!")
}
