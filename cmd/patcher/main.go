// Utility script that can be used to patch unpacked PSOBB executables
// in order to force them to connect to a different IP address.
//
// For CLI usage instructions:
//     patcher -help
//
// Before running the tool you may need to uncomment one of the offset
// blocks at the top (or define your own). These offsets are the addresses
// of the hardcoded IP addresses to which the client will attempt to connect,
// they can be found with a hex editor.
package main

import (
	"flag"
	"fmt"
	"os"
)

const blockLen = 0x18

// Use this block instead if you're using TethVer12510 executables.
//var offsets = []int64{
//	0x56b8eC,
//	0x56B904,
//	0x56B930,
//	0x56B94C,
//	0x56B968,
//	0x56B984,
//}

// Use this block instead if you're using TethVer12513 executables.
//var offsets = []int64{
//	0x56d70c,
//	0x56d724,
//	0x56d76c,
//	0x56d788,
//	0x56d7a4,
//}

// Use this block instead if you're using the Ephinea executables.
var offsets = []int64{
	0x56D70C,
	0x56D724,
	0x56D750,
	0x56D76C,
	0x56D788,
}

var exePath = flag.String("exe", "Psobb.exe", "Path (full or relative) to the PSOBB executable")
var newAddress = flag.String("address", "127.0.0.1", "The new address or IPv4 address")

func main() {
	flag.Parse()

	if len(*newAddress) > blockLen {
		fmt.Printf("newAddress must be less than %d bytes long\n", blockLen)
		os.Exit(1)
	}

	fmt.Printf("patching exe with new address: %s\n", *newAddress)

	file, err := os.OpenFile(*exePath, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("error opening file: " + err.Error())
		os.Exit(1)
	}

	replacementAddress := make([]byte, blockLen)
	copy(replacementAddress, []byte(*newAddress)[:])

	for _, off := range offsets {
		originalAddr := make([]byte, blockLen)
		file.ReadAt(originalAddr, off)

		_, err := file.WriteAt(replacementAddress, off)
		if err != nil {
			panic(err)
		}

		fmt.Printf("replacing %s with %s\n", originalAddr, replacementAddress)
	}
}
