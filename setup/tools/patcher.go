/*
 * Utility script that can be used to patch unpacked PSOBB executables
 * in order to force them to connect to a different IP address.
 *
 * Usage: go run patcher.go ip address [exe name]
 */
package main

import (
	"fmt"
	"os"
)

const blockLen = 0x18

/* Use this block instead if you're using TethVer12510 executables.
var exeName = "Vista.exe"
var offsets = []int64{
	0x56b8eC,
	0x56B904,
	0x56B930,
	0x56B94C,
	0x56B968,
	0x56B984,
}
*/

// Offsets for the TethVer12513 executables.
var exeName = "Psobb.exe"
var offsets = []int64{
	0x56d70c,
	0x56d724,
	0x56d76c,
	0x56d788,
	0x56d7a4,
}

func main() {
	numArgs := len(os.Args)
	if numArgs < 2 {
		fmt.Println("Usage: patcher.go newhost [exe_name]")
		fmt.Println("newhost can be any IPv4 address or hostname under 24 bytes long")
		fmt.Println("Example: patcher.go localhost")
		fmt.Println("Example: patcher.go 10.0.1.2 PSOBB_Localhost.exe")
		os.Exit(0)
	}
	newIP := os.Args[1]
	if len(newIP) > blockLen {
		fmt.Printf("Hostname cannot have a length greater than 24\n")
		os.Exit(1)
	}
	if numArgs > 2 {
		exeName = os.Args[2]
	}

	repl := make([]byte, blockLen)
	bstr := []byte(newIP)
	copy(repl, bstr[:])

	fmt.Printf("Patching exe with new hostname: %s\n", newIP)
	file, err := os.OpenFile(exeName, os.O_RDWR, 0666)
	if err != nil {
		fmt.Println("Error opening file: " + err.Error())
		os.Exit(1)
	}

	test := make([]byte, blockLen)
	for _, off := range offsets {
		_, err := file.WriteAt(repl, off)
		if err != nil {
			panic(err)
		}
		file.ReadAt(test, off)

		// fmt.Printf("%X (%v bytes): ", off, blockLen)
		// for i := 0; i < blockLen; i++ {
		// 	fmt.Printf("%x", test[i])
		// }
		// fmt.Println()
	}
}
