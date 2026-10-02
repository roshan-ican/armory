package services

import (
	"context"
	"log"
)

type Signal string

const (
	SignalCorrect Signal = "correct"
	SignalWrong   Signal = "wrong"
)

type Hardware interface {
	Unlock(ctx context.Context, lockerIP string, slotNo int64) error
	Signal(ctx context.Context, lockerIP string, signal Signal) error
	Alarm(ctx context.Context, lockerIP string, on bool) error
}

type LogHardware struct{}

func (LogHardware) Unlock(ctx context.Context, lockerIP string, slotNo int64) error {
	log.Printf("hardware: unlock locker %s, slot %d", lockerIP, slotNo)
	return nil
}

func (LogHardware) Signal(ctx context.Context, lockerIP string, signal Signal) error {
	log.Printf("hardware: signal %q to locker %s", signal, lockerIP)
	return nil
}

func (LogHardware) Alarm(ctx context.Context, lockerIP string, on bool) error {
	log.Printf("hardware: wrong gun alarm %t on locker %s", on, lockerIP)
	return nil
}
