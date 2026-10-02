package services

import (
	"bytes"
	"context"
	"io"
	"log"
	"net"
	"os"
	"testing"
	"time"
)

func TestCrc8MaximMatchesBoardFrames(t *testing.T) {
	for _, tt := range []struct {
		data []byte
		want byte
	}{
		{[]byte{2, 2, 0}, 0xde},
		{[]byte{2, 0, 1}, 0x11},
		{[]byte{0, 0, 0}, 0x00},
	} {
		if got := crc8Maxim(tt.data); got != tt.want {
			t.Errorf("crc8Maxim(% x) = %#x, want %#x", tt.data, got, tt.want)
		}
	}
}

func TestEncodeFrame(t *testing.T) {
	for name, tt := range map[string]struct {
		seq, open, buzzer byte
		want              []byte
	}{
		"keep alive":  {0, 0, 0, []byte{0x24, 0, 0, 0, 0x00, 0x23}},
		"open":        {1, 1, 0, []byte{0x24, 1, 1, 0, 0x6f, 0x23}},
		"correct":     {1, 0, 1, []byte{0x24, 1, 0, 1, 0xf5, 0x23}},
		"wrong":       {2, 0, 2, []byte{0x24, 2, 0, 2, 0xf3, 0x23}},
		"high number": {255, 1, 0, []byte{0x24, 0xff, 1, 0, 0x16, 0x23}},
	} {
		if got := encodeFrame(tt.seq, tt.open, tt.buzzer); !bytes.Equal(got, tt.want) {
			t.Errorf("%s: got % x, want % x", name, got, tt.want)
		}
	}
}

func TestOutputsTake(t *testing.T) {
	keepAlive := []byte{0x24, 0, 0, 0, 0x00, 0x23}
	now := time.Now()

	t.Run("an idle or unknown board gets the keep alive", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		frames := o.take("10.0.0.5", now)
		if len(frames) != 1 || !bytes.Equal(frames[0], keepAlive) {
			t.Fatalf("got % x", frames)
		}
	})

	t.Run("a command is sent three times with one sequence number then stops", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		o.PulseUnlock("10.0.0.5")
		want := encodeFrame(1, openLock, 0)
		for i := 0; i < commandSends; i++ {
			frames := o.take("10.0.0.5", now)
			if len(frames) != 1 || !bytes.Equal(frames[0], want) {
				t.Fatalf("send %d: got % x, want % x", i+1, frames, want)
			}
		}
		frames := o.take("10.0.0.5", now)
		if len(frames) != 1 || !bytes.Equal(frames[0], keepAlive) {
			t.Fatalf("after the repeats it must go back to keep alive, got % x", frames)
		}
	})

	t.Run("each new command gets the next number and two can go out together", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		o.PulseUnlock("10.0.0.5")
		o.PulseRight("10.0.0.5")
		frames := o.take("10.0.0.5", now)
		if len(frames) != 2 || !bytes.Equal(frames[0], encodeFrame(1, openLock, 0)) || !bytes.Equal(frames[1], encodeFrame(2, 0, buzzCorrect)) {
			t.Fatalf("got % x", frames)
		}
	})

	t.Run("boards do not share numbers", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		o.PulseUnlock("10.0.0.5")
		o.PulseUnlock("10.0.0.6")
		if got := o.take("10.0.0.6", now); !bytes.Equal(got[0], encodeFrame(1, openLock, 0)) {
			t.Fatalf("got % x", got)
		}
	})

	t.Run("the sequence number wraps past 255 and skips 0", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		log.SetOutput(io.Discard)
		defer log.SetOutput(os.Stderr)
		for i := 0; i < 256; i++ {
			o.PulseRight("10.0.0.5")
		}
		if got := o.get("10.0.0.5").seq; got != 1 {
			t.Fatalf("seq = %d, want 1 after 256 commands", got)
		}
	})

	t.Run("a wrong gun sounds at once and again every few seconds until it is back", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		now := time.Now()
		o.SetWrong("10.0.0.5", true)
		first := o.take("10.0.0.5", now)
		if len(first) != 1 || !bytes.Equal(first[0], encodeFrame(1, 0, buzzWrong)) {
			t.Fatalf("got % x", first)
		}
		o.take("10.0.0.5", now)
		o.take("10.0.0.5", now)
		if got := o.take("10.0.0.5", now.Add(wrongRepeat/2)); !bytes.Equal(got[0], keepAlive) {
			t.Fatalf("it must wait before sounding again, got % x", got)
		}
		again := o.take("10.0.0.5", now.Add(wrongRepeat+100*time.Millisecond))
		if len(again) != 1 || !bytes.Equal(again[0], encodeFrame(2, 0, buzzWrong)) {
			t.Fatalf("got % x", again)
		}
		o.SetWrong("10.0.0.5", false)
		o.take("10.0.0.5", now.Add(wrongRepeat+100*time.Millisecond))
		o.take("10.0.0.5", now.Add(wrongRepeat+100*time.Millisecond))
		if got := o.take("10.0.0.5", now.Add(3*wrongRepeat)); !bytes.Equal(got[0], keepAlive) {
			t.Fatalf("once the gun is back it must stop, got % x", got)
		}
	})

	t.Run("asking again while it is already wrong does not start a second alarm", func(t *testing.T) {
		o := NewOutputs(nil, 0, nil)
		o.SetWrong("10.0.0.5", true)
		o.SetWrong("10.0.0.5", true)
		if got := o.get("10.0.0.5").seq; got != 1 {
			t.Fatalf("seq = %d, want 1", got)
		}
	})
}

func TestOutputsRunSendsFramesFromTheSharedSocket(t *testing.T) {
	board, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer board.Close()
	out, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()

	port := board.LocalAddr().(*net.UDPAddr).Port
	o := NewOutputs(out, port, func(context.Context) ([]string, error) { return []string{"127.0.0.1"}, nil })
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go o.Run(ctx)

	read := func() []byte {
		buf := make([]byte, 16)
		board.SetReadDeadline(time.Now().Add(2 * time.Second))
		n, from, err := board.ReadFrom(buf)
		if err != nil {
			t.Fatalf("no frame received: %v", err)
		}
		if from.String() != out.LocalAddr().String() {
			t.Fatalf("sent from %v, want the shared socket %v", from, out.LocalAddr())
		}
		return buf[:n]
	}

	keepAlive := []byte{0x24, 0, 0, 0, 0x00, 0x23}
	if got := read(); !bytes.Equal(got, keepAlive) {
		t.Fatalf("an idle board must get the keep alive, got % x", got)
	}

	o.PulseUnlock("127.0.0.1")
	want := encodeFrame(1, openLock, 0)
	saw := false
	for i := 0; i < 4 && !saw; i++ {
		saw = bytes.Equal(read(), want)
	}
	if !saw {
		t.Fatal("the open command never arrived")
	}
}
