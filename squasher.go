package squasher

import (
	"errors"
	"fmt"
	"sync"
)

type Squasher struct {
	*sync.Mutex
	circle      []byte
	start_value int64
	start_byte  uint
	start_bit   uint
	nextchan    chan int64
	latest      int64
}

func NewSquasher(start int64, size int32) *Squasher {
	if size < 2 {
		size = 2
	}
	circlelen := (size + 7) / 8 // number of byte to store <size> bit
	circlelen++                 // add extra byte to mark end of circle
	circle := make([]byte, circlelen)
	circle[0] = 1
	return &Squasher{
		Mutex:       &sync.Mutex{},
		nextchan:    make(chan int64, 0),
		start_value: start,
		start_byte:  0,
		start_bit:   0,
		circle:      circle,
	}
}

func closeCircle(circle []byte, start_byte uint) {
	ln := uint(len(circle))
	if ln < 1 { // need at least 2 byte to form a circle
		return
	}

	start_byte %= ln
	prevbyte := start_byte - 1
	if start_byte == 0 {
		prevbyte = ln - 1
	}
	circle[prevbyte] = 0
}

func (s *Squasher) Mark(i int64) error {
	s.Lock()
	defer s.Unlock()
	if i > s.latest {
		s.latest = i
	}
	dist := i - s.start_value
	if dist <= 0 {
		return nil
	}
	ln := uint(len(s.circle))

	if int64(ln)*8-1 < dist {
		return errors.New(fmt.Sprintf("out of range, i should be less than %d", int64(ln)-1+s.start_value))
	}

	bytediff := uint((dist + int64(s.start_bit)) / 8)
	nextbyte := (s.start_byte + bytediff) % ln

	bit := uint((dist + int64(s.start_bit)) % 8)
	s.circle[nextbyte] |= 1 << bit
	if dist != 1 {
		return nil
	}

	s.start_value, s.start_byte, s.start_bit =
		getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)

	closeCircle(s.circle, s.start_byte)
	s.nextchan <- s.start_value
	return nil
}

// getFirstZeroBit return the first zero bit
func getFirstZeroBit(x byte) uint {
	for bit := uint(0); bit < 8; bit++ {
		if x%2 == 0 {
			return bit
		}
		x >>= 1
	}
	return 8
}

func getNextNonFFByte(circle []byte, start_byte uint) uint {
	ln := uint(len(circle))
	if ln == 0 {
		return 0
	}
	start_byte = start_byte % ln
	byt := start_byte
	for circle[byt] == 0xFF {
		byt = (byt + 1) % ln
		if byt == start_byte { // finish one loop
			break
		}
	}
	return byt
}

func getNextMissingIndex(circle []byte, start_value int64, start_byte, start_bit uint) (int64, uint, uint) {
	ln := uint(len(circle))
	if ln == 0 {
		return 0, 0, 0
	}
	byt := getNextNonFFByte(circle, start_byte)
	bit := getFirstZeroBit(circle[byt])
	if bit == 8 || bit == 0 { // got 0xFF
		if byt == 0 {
			byt = ln
		}
		byt--
		bit = 8
	}
	bit--
	if byt < start_byte {
		byt += ln
	}
	dist := (byt-start_byte)*8 + bit - start_bit
	return start_value + int64(dist), byt % ln, bit
}

func (s *Squasher) Next() <-chan int64 { return s.nextchan }

func (s *Squasher) GetStatus() string {
	ofs, _, _ := getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)
	return fmt.Sprintf("first offset: %d, last offset: %d, next missing %d", s.start_value, s.latest, ofs)
}
