// Utility script that can be used to patch unpacked PSOBB executables
// in order to force them to connect to a different IP address.
//
// For CLI usage instructions:
//
//	patcher -help
//
// Before running the tool you may need to uncomment one of the offset
// blocks at the top (or define your own). These offsets are the addresses
// of the hardcoded IP addresses to which the client will attempt to connect,
// they can be found with a hex editor.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var patchCmd = &cobra.Command{
	Use:   "patcher [exe]",
	Short: "Updates the IP address in PSOBB executables",
	Run:   PatchCommand,
	Args:  cobra.ExactArgs(1),
}

const blockLen = 0x18

var (
	NewAddressFlag string
	ExeVersionFlag string
)

func PatchCommand(cmd *cobra.Command, args []string) {
	var offsets []int64
	switch ExeVersionFlag {
	case "TethVer12510":
		// Use this block instead if you're using TethVer12510 executables.
		offsets = []int64{
			0x56b8eC,
			0x56B904,
			0x56B930,
			0x56B94C,
			0x56B968,
			0x56B984,
		}
	case "TethVer12513":
		// Use this block instead if you're using TethVer12513 executables.
		offsets = []int64{
			0x56d70c,
			0x56d724,
			0x56d76c,
			0x56d788,
			0x56d7a4,
		}
	default:
		// Use this block instead if you're using the Ephinea executables.
		offsets = []int64{
			0x56D70C,
			0x56D724,
			0x56D750,
			0x56D76C,
			0x56D788,
		}
	}

	if len(NewAddressFlag) > blockLen {
		fmt.Printf("newAddress must be less than %d bytes long\n", blockLen)
		os.Exit(1)
	}

	fmt.Printf("patching exe with new address: %s\n", NewAddressFlag)

	file, err := os.OpenFile(args[0], os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("error opening file: " + err.Error())
		os.Exit(1)
	}

	replacementAddress := make([]byte, blockLen)
	copy(replacementAddress, []byte(NewAddressFlag)[:])

	for _, off := range offsets {
		originalAddr := make([]byte, blockLen)
		if _, err := file.ReadAt(originalAddr, off); err != nil {
			fmt.Printf("failed to read byte at %x, error: %v", off, err)
			return
		}

		if _, err := file.WriteAt(replacementAddress, off); err != nil {
			fmt.Printf("failed to write byte at %x, error: %v", off, err)
			return
		}

		fmt.Printf("replacing %s with %s\n", originalAddr, replacementAddress)
	}
}
