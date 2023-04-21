package bytes

import (
	"reflect"
	"testing"

	"github.com/dcrodman/archon/internal/packets"
	"github.com/google/go-cmp/cmp"
)

func TestConvertToUtf16(t *testing.T) {
	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "empty string",
			args: args{
				str: "",
			},
			want: []byte{},
		},
		{
			name: "arbitrary text",
			args: args{
				str: "Archon Server",
			},
			want: []byte{65, 0, 114, 0, 99, 0, 104, 0, 111, 0, 110, 0, 32, 0, 83, 0, 101, 0, 114, 0, 118, 0, 101, 0, 114, 0},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertToUtf16(tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ConvertToUtf16() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStripPadding(t *testing.T) {
	type args struct {
		b []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "does not alter strings without padding",
			args: args{
				b: []byte{117, 115, 101, 114, 110, 97, 109, 101},
			},
			want: []byte{117, 115, 101, 114, 110, 97, 109, 101},
		},
		{
			name: "removes trailing padding",
			args: args{
				b: []byte{117, 115, 101, 114, 110, 97, 109, 101, 0, 0, 0, 0},
			},
			want: []byte("username"),
		},
		{
			name: "removes all padding",
			args: args{
				b: []byte{0, 0, 0, 0, 0},
			},
			want: []byte{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripPadding(tt.args.b); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("StripPadding() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStructConversions(t *testing.T) {
	command := []byte{
		0x4c, 0x00, 0x02, 0x00, 0x50, 0x61, 0x74, 0x63, 0x68, 0x20, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
		0x2e, 0x20, 0x43, 0x6f, 0x70, 0x79, 0x72, 0x69, 0x67, 0x68, 0x74, 0x20, 0x53, 0x6f, 0x6e, 0x69,
		0x63, 0x54, 0x65, 0x61, 0x6d, 0x2c, 0x20, 0x4c, 0x54, 0x44, 0x2e, 0x20, 0x32, 0x30, 0x30, 0x31,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x4c, 0x83, 0x58, 0x5e, 0xec, 0xd6, 0x0c, 0x7f,
	}

	var welcomePacket packets.PatchWelcome
	StructFromBytes(command, &welcomePacket)

	if diff := cmp.Diff(welcomePacket.Copyright[:], []byte("Patch Server. Copyright SonicTeam, LTD. 2001")); diff != "" {
		t.Errorf("welcome packet Copyright did not match expected, diff:\n%s", diff)
	}

	convertedPacket, bytes := BytesFromStruct(welcomePacket)
	if bytes != len(command) {
		t.Errorf("expected bytes to equal the length of the packet (%d), got = %v", convertedPacket, bytes)
	}

	if diff := cmp.Diff(command, convertedPacket); diff != "" {
		t.Errorf("expected converted packet to match original. diff:\n%s", diff)
	}
}
