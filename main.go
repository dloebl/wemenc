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
	
	if err := encodeToWEM(*inputFlag, *outputFlag, *bitrateFlag); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Println("Done!")
}
