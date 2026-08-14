package shipgate

import (
	"bytes"
	"encoding/binary"
	"reflect"
)

// These implementations are duplicated from bytes.go both for convenience and also to prevent any
// changes there from having unintended consequences on the way bytes are stored in the database.

func UnmarshalStruct(data []byte, targetStruct interface{}) {
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

func MarshalStruct(data interface{}) ([]byte, int) {
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
			b, _ := MarshalStruct(field.Interface())
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
