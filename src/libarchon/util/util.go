// Functions and constants shared between server components.
package util

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
)

const ServerConfigDir = "/usr/local/share/archon"
const displayWidth = 16

type ServerError struct {
	Message string
}

func (err ServerError) Error() string {
	return err.Message
}

// Sets the values of a slice of bytes (up to length) to 0.
func ZeroSlice(arr []byte, length int) {
	if arrLen := len(arr); arrLen < length {
		length = arrLen
	}
	for i := 0; i < length; i++ {
		arr[i] = 0
	}
}

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
	for i := length; i < displayWidth; i++ {
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
func PrintPayload(data []uint8, pktLen int) {
	for rem, offset := pktLen, 0; rem > 0; rem -= displayWidth {
		if rem < displayWidth {
			printPacketLine(data[(pktLen-rem):pktLen], rem, offset)
		} else {
			printPacketLine(data[offset:offset+displayWidth], displayWidth, offset)
		}
		offset += displayWidth
	}
}

// Serializes the fields of a struct to an array of bytes in the order in which the fields are
// declared. Calls panic() if data is not a struct or pointer to struct.
func BytesFromStruct(data interface{}) []byte {
	val := reflect.ValueOf(data)
	valKind := val.Kind()
	if valKind == reflect.Ptr {
		val = reflect.ValueOf(data).Elem()
		valKind = val.Kind()
	}

	if valKind != reflect.Struct {
		panic("data must of type struct or ptr to struct, got: " + valKind.String())
	}

	bytes := new(bytes.Buffer)
	/* Keeping my original implementation here just in case since it's capable of working
	with types of dynamic size, unlike the binary.Write function.
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
	*/
	binary.Write(bytes, binary.LittleEndian, val.Interface())
	return bytes.Bytes()
}

// Populates the struct pointed to by targetStructby reading in a stream of bytes and filling the values
// in sequential order. Note that the struct itself must be of fixed width; dynamic types will result
// in mistranslated values (or possibly a panic).
func StructFromBytes(data []byte, targetStruct interface{}) {
	if kind := reflect.TypeOf(targetStruct).Kind(); kind != reflect.Ptr {
		panic("targetStruct must be a ptr to struct, got: " + kind.String())
	}
	reader := bytes.NewReader(data)
	binary.Read(reader, binary.LittleEndian, targetStruct)
}
