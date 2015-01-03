// Shared methods and operations common to each of the server components.
package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
)

const DISPLAY_WIDTH = 16

func printPacketLine(data []uint8, length int, offset int) {
	fmt.Printf("%04x ", offset)
	// Print our bytes.
	for i, j := 0, 0; i < length; i++ {
		if j == 8 {
			// Visual aid - spacing between groups of 8 bytes.
			j = 0
			fmt.Print("  ")
		}
		fmt.Printf("%02x ", data[i])
		j++
	}
	// Fill in the gap if we don't have enough bytes to fill the line.
	for i := length; i < DISPLAY_WIDTH; i++ {
		if i == 8 {
			fmt.Print("  ")
		}
		fmt.Print("   ")
	}
	fmt.Print("    ")
	// Display the print characters as-is, others as periods.
	for i := 0; i < length; i++ {
		c := data[i]
		if strconv.IsPrint(rune(c)) {
			fmt.Printf("%c", data[i])
		} else {
			fmt.Print(".")
		}
	}
	fmt.Println()
}

// Print the contents of a packet to stdout in two columns, one for bytes and the other
// for their ascii representation.
func PrintPayload(data []uint8) {
	pktLen := len(data)
	for rem, offset := pktLen, 0; rem > 0; rem -= DISPLAY_WIDTH {
		if rem < DISPLAY_WIDTH {
			printPacketLine(data[(pktLen-rem):pktLen], rem, offset)
		} else {
			printPacketLine(data[offset:offset+DISPLAY_WIDTH], DISPLAY_WIDTH, offset)
		}
		offset += DISPLAY_WIDTH
	}
}

// Serializes the fields of a struct to an array of bytes in the order in which the fields are
// declared. Calls panic() if data is not a struct or pointer to struct.
func StructToBytes(data interface{}) []byte {
	val := reflect.ValueOf(data)
	valKind := val.Kind()
	if valKind == reflect.Ptr {
		val = reflect.ValueOf(data).Elem()
		valKind = val.Kind()
	}

	if valKind != reflect.Struct {
		panic("data must of type struct or struct ptr, got: " + valKind.String())
	}

	bytes := new(bytes.Buffer)
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		switch kind := field.Kind(); kind {
		case reflect.Struct:
			binary.Write(bytes, binary.LittleEndian, StructToBytes(field.Interface()))
		case reflect.Array, reflect.Slice:
			binary.Write(bytes, binary.LittleEndian, field.Interface())
		case reflect.Uint8:
			binary.Write(bytes, binary.LittleEndian, uint8(field.Uint()))
		case reflect.Uint16:
			binary.Write(bytes, binary.LittleEndian, uint16(field.Uint()))
		case reflect.Uint32:
			binary.Write(bytes, binary.LittleEndian, uint32(field.Uint()))
		case reflect.Uint, reflect.Uint64:
			binary.Write(bytes, binary.LittleEndian, field.Uint())
		case reflect.Int8:
			binary.Write(bytes, binary.LittleEndian, int8(field.Int()))
		case reflect.Int16:
			binary.Write(bytes, binary.LittleEndian, int16(field.Int()))
		case reflect.Int32:
			binary.Write(bytes, binary.LittleEndian, int32(field.Int()))
		case reflect.Int, reflect.Int64:
			binary.Write(bytes, binary.LittleEndian, field.Int())
		}
	}
	return bytes.Bytes()
}
