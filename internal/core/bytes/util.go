package bytes

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"unicode/utf16"
)

// ConvertToUtf16 converts a UTF-8 string to UTF-16 LE and return it as an array of bytes.
func ConvertToUtf16(str string) []byte {
	strRunes := bytes.Runes([]byte(str))
	encoded := utf16.Encode(strRunes)

	// Convert the array of UTF-16 elements to a slice of uint8 elements in
	// little endian order. E.g: [0x1234] -> [0x34, 0x12]
	expanded := make([]uint8, 2*len(encoded))
	for i, v := range encoded {
		idx := i * 2
		expanded[idx] = uint8(v)
		expanded[idx+1] = uint8((v >> 8) & 0xFF)
	}
	return expanded
}

// StripPadding returns a slice of b without the trailing 0s.
func StripPadding(b []byte) []byte {
	for i := len(b) - 1; i >= 0; i-- {
		if b[i] != 0 {
			return b[:i+1]
		}
	}
	return []byte{}
}

// BytesFromStruct serializes the fields of a struct to an array of bytes in the
// order in which the fields are declared and returns total number of bytes converted.
// Panics if data is not a struct or pointer to struct, or if there was an error writing a field.
func BytesFromStruct(data interface{}) ([]byte, int) {
	val := reflect.ValueOf(data)
	valKind := val.Kind()

	if valKind == reflect.Ptr {
		val = reflect.ValueOf(data).Elem()
		valKind = val.Kind()
	}

	if valKind != reflect.Struct {
		panic("BytesFromStruct(): data must of type struct " +
			"or ptr to struct, got: " + valKind.String())
	}

	convertedBytes := new(bytes.Buffer)
	// It's possible to use binary.Write on val.Interface itself, but doing
	// so prevents this function from working with dynamically sized types.
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		var err error
		switch kind := field.Kind(); kind {
		case reflect.Struct, reflect.Ptr:
			b, _ := BytesFromStruct(field.Interface())
			err = binary.Write(convertedBytes, binary.LittleEndian, b)
		default:
			err = binary.Write(convertedBytes, binary.LittleEndian, field.Interface())
		}
		if err != nil {
			panic(err.Error())
		}
	}
	return convertedBytes.Bytes(), convertedBytes.Len()
}

// StructFromBytes populates the struct pointed to by targetStruct by reading in a
// stream of bytes and filling the values in sequential order.
func StructFromBytes(data []byte, targetStruct interface{}) {
	targetVal := reflect.ValueOf(targetStruct)

	if valKind := targetVal.Kind(); valKind != reflect.Ptr {
		panic("StructFromBytes(): targetStruct must be a " +
			"ptr to struct, got: " + valKind.String())
	}

	reader := bytes.NewReader(data)
	val := targetVal.Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)

		var err error
		switch field.Kind() {
		case reflect.Ptr:
			err = binary.Read(reader, binary.LittleEndian, field.Interface())
		default:
			err = binary.Read(reader, binary.LittleEndian, field.Addr().Interface())
		}
		if err != nil {
			panic(err.Error())
		}
	}
}

//const displayWidth = 16
//
//// PrintPayload prints the contents of a packet to stdout in two columns, one for bytes and
//// the other for their ascii representation.
//func PrintPayload(data []uint8, pktLen int) {
//	for rem, offset := pktLen, 0; rem > 0; rem -= displayWidth {
//		if rem < displayWidth {
//			printPacketLine(data[(pktLen-rem):pktLen], rem, offset)
//		} else {
//			printPacketLine(data[offset:offset+displayWidth], displayWidth, offset)
//		}
//		offset += displayWidth
//	}
//}
//
//// printPacketLine writes one line of data to stdout.
//func printPacketLine(data []uint8, length int, offset int) {
//	fmt.Printf("(%04X) ", offset)
//	// Print our bytes.
//	for i, j := 0, 0; i < length; i++ {
//		if j == 8 {
//			// Visual aid - spacing between groups of 8 bytes.
//			j = 0
//			fmt.Print("  ")
//		}
//		fmt.Printf("%02x ", data[i])
//		j++
//	}
//	// Fill in the gap if we don't have enough bytes to fill the line.
//	for i := length; i < displayWidth; i++ {
//		if i == 8 {
//			fmt.Print("  ")
//		}
//		fmt.Print("   ")
//	}
//	fmt.Print("    ")
//	// Display the print characters as-is, others as periods.
//	for i := 0; i < length; i++ {
//		c := data[i]
//		if strconv.IsPrint(rune(c)) {
//			fmt.Printf("%c", data[i])
//		} else {
//			fmt.Print(".")
//		}
//	}
//	fmt.Println()
//}
