package squasher

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type Squasher struct {
	*sync.Mutex
	circle      []byte
	start_value int64
	start_byte  uint
	start_bit   uint
	mynextchan  chan int64 // internal next chan
	nextchan    chan int64 // next chan receive next value
	latest      int64
}

// a circle with size = 2
//     start_byte: 1 --v     v-- start_bit: 3
// byt 0 0 0 0 0 0 0 0 1 1 1 1 1 1 1 1 2 2 2 2 2 2 2 2 3 3 3 3 3 3 3 3
// bit 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7 0 1 2 3 4 5 6 7

//
//
func NewSquasher(start int64, size int32) *Squasher {
	if size < 2 {
		size = 2
	}
	circlelen := size/8 + 1 // number of byte to store <size> bit
	circlelen++             // add extra byte to mark end of circle
	circle := make([]byte, circlelen, circlelen)
	circle[0] = 1
	s := &Squasher{
		Mutex:       &sync.Mutex{},
		nextchan:    make(chan int64),
		mynextchan:  make(chan int64),
		start_value: start - 1,
		start_byte:  0,
		start_bit:   0,
		circle:      circle,
	}
	go s.run()
	return s
}

// closeCircle set the end_byte (byte before start_byte) to zero
func closeCircle(circle []byte, start_byte uint) {
	ln := uint(len(circle))
	prevbyte := (start_byte + ln - 1) % ln
	circle[prevbyte] = 0
}

//
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

	if uint(dist) > ln*8-1 {
		return fmt.Errorf("out of range, i should be less than %d", int64(ln)-1+s.start_value)
	}

	bytediff := (uint(dist) + s.start_bit) / 8
	nextbyte := (s.start_byte + bytediff) % ln

	bit := (uint(dist) + s.start_bit) % 8
	s.circle[nextbyte] |= 1 << bit
	if dist != 1 {
		return nil
	}
	s.start_value, s.start_byte, s.start_bit =
		getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)

	closeCircle(s.circle, s.start_byte)
	s.mynextchan <- s.start_value
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

func (s *Squasher) run() {
	mt := &sync.Mutex{}
	curval := int64(math.MaxInt64) // init val

	go func() {
		lastv := int64(math.MaxInt64)
		for {
			mt.Lock()
			if curval == math.MaxInt64 || lastv == curval { // no new value
				mt.Unlock()
				time.Sleep(1 * time.Second)
				continue
			}
			lastv = curval // new value have been set
			mt.Unlock()
			s.nextchan <- lastv
		}
	}()

	for v := range s.mynextchan {
		mt.Lock()
		curval = v
		mt.Unlock()
	}
}

// get next non 0xFF byte, assume that the circle alway end with 0x00
func getNextNonFFByte(circle []byte, start_byte uint) uint {
	ln := uint(len(circle))
	for ; circle[start_byte%ln] == 0xFF; start_byte++ {
		if start_byte > 2*ln {
			panic(fmt.Sprintf("invalid circle, no end: %v", circle))
		}
	}
	return start_byte % ln
}

func getNextMissingIndex(circle []byte, start_value int64, start_byte, start_bit uint) (int64, uint, uint) {
	ln := uint(len(circle))
	byt := getNextNonFFByte(circle, start_byte)
	bit := getFirstZeroBit(circle[byt])
	if bit == 0 { // got 0 -> decrease 1 byte
		byt = (byt + ln - 1) % ln
	}
	bit = (bit + 8 - 1) % 8

	if byt < start_byte {
		byt += ln
	}
	dist := (byt-start_byte)*8 + bit - start_bit
	return start_value + int64(dist), byt % ln, bit
}

func (s *Squasher) Next() <-chan int64 { return s.nextchan }

func (s *Squasher) GetStatus() string {
	ofs, _, _ := getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)
	return fmt.Sprintf("[%d .. %d .. %d]", s.start_value, ofs, s.latest)
}
