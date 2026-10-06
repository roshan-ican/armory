package sensor

import (
	"bytes"
	"errors"
	"testing"
)

func TestDecodeSensorFrame(t *testing.T) {
	tests := []struct {
		name    string
		in      []byte
		want    []byte
		wantErr error
	}{
		{"real packet 1", []byte{0x24, 0x02, 0x02, 0x00, 0xDE, 0x23}, []byte{0, 2, 2}, nil},
		{"real packet 2", []byte{0x24, 0x02, 0x00, 0x01, 0x11, 0x23}, []byte{1, 0, 2}, nil},
		{"all present", []byte{0x24, 0x01, 0x01, 0x01, 0x31, 0x23}, []byte{1, 1, 1}, nil},
		{"only wire slot 0 present", []byte{0x24, 0x01, 0x00, 0x00, 0x00, 0x23}, []byte{0, 0, 1}, nil},
		{"too short", []byte{0x24, 0x02, 0x00, 0x01, 0x11}, nil, errFrameLength},
		{"too long", []byte{0x24, 0x02, 0x00, 0x01, 0x11, 0x23, 0x00}, nil, errFrameLength},
		{"empty", []byte{}, nil, errFrameLength},
		{"bad header", []byte{0x25, 0x02, 0x00, 0x01, 0x11, 0x23}, nil, errFrameBytes},
		{"bad tail", []byte{0x24, 0x02, 0x00, 0x01, 0x11, 0x24}, nil, errFrameBytes},
		{"invalid state slot 0", []byte{0x24, 0x03, 0x01, 0x01, 0x00, 0x23}, nil, errSlotState},
		{"invalid state slot 1", []byte{0x24, 0x01, 0x03, 0x00, 0x00, 0x23}, nil, errSlotState},
		{"invalid state slot 2", []byte{0x24, 0x01, 0x01, 0xFF, 0x00, 0x23}, nil, errSlotState},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodeSensorFrame(tc.in)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("err = %v, want %v", err, tc.wantErr)
			}
			if !bytes.Equal(got, tc.want) {
				t.Fatalf("got % x, want % x", got, tc.want)
			}
		})
	}
}

func TestDecodeSensorFrameCopiesSlots(t *testing.T) {
	in := []byte{0x24, 0x01, 0x01, 0x01, 0x31, 0x23}
	got, err := decodeSensorFrame(in)
	if err != nil {
		t.Fatal(err)
	}
	in[3] = 0x00
	if got[0] != 0x01 {
		t.Fatalf("result changed when input buffer was reused: % x", got)
	}
}
