package squasher

import (
	"sync"
)

type Squasher struct {
	circle []byte
	start_value int64
	start_byte uint
	start_bit uint
	lock *sync.Mutex
	nextchan chan int64
}

func NewSquasher(start int64, size int32) *Squasher {
	circlelen := size / 8 + 1
	return &Squasher{
		lock: &sync.Mutex{},
		nextchan: make(chan int64, 0),
		start_value: start,
		start_byte: 0,
		start_bit: 0,
		circle: make([]byte, circlelen),
	}
}

func (s *Squasher) Mark(i int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	dist := i - s.start_value
	if dist < 0 {
		return nil
	}

	nextbyte := uint((dist + int64(s.start_bit - 8)) / int64(8))
	// 8 - t + 8 * diffbyte + x = dist

	bit := uint(dist - int64(8 + s.start_bit) - int64(8 * nextbyte))

	ind := nextbyte + s.start_byte % uint(len(s.circle))
	s.circle[ind] |= 1 << bit

	if dist == 1 {
		s.start_value, s.start_byte, s.start_bit = getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)
		s.nextchan <- s.start_value
	}
	return nil
}

// getFirstZeroBit return the first zero bit
func getFirstZeroBit(x byte) uint {
	for bit := uint(0); bit < 8; bit++ {
		if x % 2 == 0 {
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
	// bit never be 8 because circle[byt] is non 0xFF
	if bit == 0 {
		bit, byt = 7, byt - 1
	} else {
		bit--
	}

	dist := (byt + ln - start_byte) % ln * 8 + bit - start_bit
	return start_value + int64(dist), byt, bit
}

func (s *Squasher) Next() <-chan int64 {
	return s.nextchan
}
