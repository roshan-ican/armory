package services

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"
)

const (
	outputHeader = 0x24
	outputTail   = 0x23
	sendEvery    = 500 * time.Millisecond
	commandSends = 3
	wrongRepeat  = 3 * time.Second
)

const (
	openLock    = 1
	buzzCorrect = 1
	buzzWrong   = 2
)

func crc8Maxim(data []byte) byte {
	var crc byte

	for _, b := range data {
		crc ^= b
		for i := 0; i < 8; i++ {
			if crc&1 != 0 {
				crc = crc>>1 ^ 0x8C
			} else {
				crc >>= 1
			}
		}
	}
	return crc
}

func encodeFrame(seq, open, buzzer byte) []byte {
	body := []byte{seq, open, buzzer}
	frame := make([]byte, 0, 6)
	frame = append(frame, outputHeader)
	frame = append(frame, body...)
	frame = append(frame, crc8Maxim(body), outputTail)
	return frame
}

type command struct {
	frame []byte
	left  int
}

type outputState struct {
	seq       byte
	pending   []command
	wrong     bool
	wrongNext time.Time
}

type Outputs struct {
	mu     sync.Mutex
	state  map[string]*outputState
	conn   net.PacketConn
	port   int
	ips    func(ctx context.Context) ([]string, error)
	doorIp string
	kick   chan struct{}
}

func NewOutputs(conn net.PacketConn, port int, ips func(ctx context.Context) ([]string, error), doorIp string) *Outputs {
	return &Outputs{
		state:  make(map[string]*outputState),
		conn:   conn,
		port:   port,
		ips:    ips,
		doorIp: doorIp,
		kick:   make(chan struct{}, 1),
	}
}

func (o *Outputs) get(ip string) *outputState {
	s, ok := o.state[ip]
	if !ok {
		s = &outputState{}
		o.state[ip] = s
	}
	return s
}

func (o *Outputs) wake() {
	select {
	case o.kick <- struct{}{}:
	default:
	}
}

func (o *Outputs) queue(ip string, open, buzzer byte, what string) {
	s := o.get(ip)
	s.seq++
	if s.seq == 0 {
		s.seq = 1
	}
	s.pending = append(s.pending, command{frame: encodeFrame(s.seq, open, buzzer), left: commandSends})
	log.Printf("outputs %s: %s seq=%d", ip, what, s.seq)
	o.wake()
}

func (o *Outputs) PulseUnlock(ip string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.queue(ip, openLock, 0, "open")
}

func (o *Outputs) PulseDoor() {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.queue(o.doorIp, openLock, 0, "door")
}

func (o *Outputs) PulseRight(ip string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.queue(ip, 0, buzzCorrect, "correct")
}

func (o *Outputs) SetWrong(ip string, on bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	s := o.get(ip)
	if on && !s.wrong {
		o.queue(ip, 0, buzzWrong, "wrong")
		s.wrongNext = time.Now().Add(wrongRepeat)
	}
	s.wrong = on
}

func (o *Outputs) take(ip string, now time.Time) [][]byte {
	o.mu.Lock()
	defer o.mu.Unlock()
	s, ok := o.state[ip]
	if !ok {
		return [][]byte{encodeFrame(0, 0, 0)}
	}
	if s.wrong && !now.Before(s.wrongNext) {
		o.queue(ip, 0, buzzWrong, "wrong again")
		s.wrongNext = now.Add(wrongRepeat)
	}
	var frames [][]byte
	kept := s.pending[:0]
	for _, c := range s.pending {
		frames = append(frames, c.frame)
		c.left--
		if c.left > 0 {
			kept = append(kept, c)
		}
	}
	s.pending = kept
	if len(frames) == 0 {
		frames = append(frames, encodeFrame(0, 0, 0))
	}
	return frames
}

func (o *Outputs) Unlock(ctx context.Context, lockerIP string, slotNo int64) error {
	o.PulseUnlock(lockerIP)
	return nil
}

func (o *Outputs) Signal(ctx context.Context, lockerIP string, signal Signal) error {
	if signal != SignalCorrect {
		return fmt.Errorf("unsupported signal %q", signal)
	}
	o.PulseRight(lockerIP)
	return nil
}

func (o *Outputs) Alarm(ctx context.Context, lockerIP string, on bool) error {
	o.SetWrong(lockerIP, on)
	return nil
}
func (o *Outputs) OpenDoor(ctx context.Context) error {
	o.PulseDoor()
	return nil
}
func (o *Outputs) Run(ctx context.Context) {
	ticker := time.NewTicker(sendEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		case <-o.kick:
		}
		o.sendAll(ctx)
	}
}

func (o *Outputs) sendAll(ctx context.Context) {
	ips, err := o.ips(ctx)
	if err != nil {
		log.Printf("outputs: list lockers: %v", err)
		return
	}
	now := time.Now()
	for _, ip := range ips {
		addr := net.ParseIP(ip)
		if addr == nil {
			continue
		}
		for _, frame := range o.take(ip, now) {
			if _, err := o.conn.WriteTo(frame, &net.UDPAddr{IP: addr, Port: o.port}); err != nil {
				log.Printf("outputs %s: send: %v", ip, err)
			}
		}
	}
}
