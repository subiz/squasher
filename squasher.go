package squasher

import (
	"fmt"
	"sync"
)

type Squasher struct {
	*sync.Mutex
	circle      []byte
	start_value int64
	start_byte  uint
	start_bit   uint
	nextchan    chan int64 // next chan receive next value
	latest      int64
}

func (s *Squasher) Print() {
	fmt.Printf("CIRCLE(%d) %d:%d=%d ------\n", len(s.circle), s.start_byte, s.start_bit, s.start_value)
	for _, c := range s.circle {
		fmt.Printf("%08b ", c)
	}
	fmt.Println("")
}

// a circle with size = 2
//     start_byte: 1 --v     v-- start_bit: 4
// byt 0 0 0 0 0 0 0 0 1 1 1 1 1 1 1 1 2 2 2 2 2 2 2 2 3 3 3 3 3 3 3 3
// bit 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0 7 6 5 4 3 2 1 0
// cir 0 0 0 0 0 0 0 0 0 0 0 1 0 0 1 0 1 0 0 0 1 1 0 1 1 1 1 1 0 1 0 0
//
//
func NewSquasher(start int64, size int32) *Squasher {
	if size < 2 {
		size = 2
	}
	size++
	circlelen := size / 8 // number of byte to store <size> bit
	if size%8 > 0 {
		circlelen++
	}

	circle := make([]byte, circlelen, circlelen)
	circle[0] = 1
	s := &Squasher{
		Mutex:       &sync.Mutex{},
		nextchan:    make(chan int64, 1),
		start_value: start - 1,
		circle:      circle,
	}
	return s
}

// closeCircle set the end_byte (byte before start_byte) to zero
func zeroCircle(circle []byte, frombyt, tobyt, frombit, tobit uint) {
	ln := uint(len(circle))
	if ln <= frombyt || ln <= tobyt {
		panic("something wrong here, from or to is greater than len of circle")
	}
	// zero out first byte
	var mask byte
	if frombyt == tobyt {
		for i := frombit; i%8 != tobit; i++ {
			mask |= 1 << (i % 8)
		}
		mask ^= 0xFF
		circle[frombyt] &= mask
		if frombit < tobit {
			return
		}
		frombyt++
	} else {
		var mask byte = (0xFF >> tobit) << tobit
		circle[tobyt] &= mask

		mask = (0xFF << (8 - frombit) >> (8 - frombit))
		circle[frombyt] &= mask
	}

	for i := frombyt; i%ln != tobyt; i++ {
		circle[i%ln] = 0
	}
}

func setBit(circle []byte, start_byte, start_bit uint, start_value, i int64) {
	ln := uint(len(circle))
	dist := i - start_value
	if dist <= 0 {
		return
	}
	bytediff := (uint(dist) + start_bit) / 8
	nextbyte := (start_byte + bytediff) % ln

	bit := (uint(dist) + start_bit) % 8
	circle[nextbyte] |= 1 << bit
}

// Mark a value i as processed
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

	if uint(dist) > (ln)*8-1 {
		return fmt.Errorf("out of range, i should be less than %d", int64(ln)*8+s.start_value)
	}

	setBit(s.circle, s.start_byte, s.start_bit, s.start_value, i)
	if dist != 1 {
		return nil
	}
	nextval, nextbyte, nextbit :=
		getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)
	zeroCircle(s.circle, s.start_byte, nextbyte, s.start_bit, nextbit)
	s.start_value, s.start_byte, s.start_bit = nextval, nextbyte, nextbit
	s.pushNext(nextval)
	return nil
}

// not thread safe, so remember to lock before call this
func (s *Squasher) pushNext(val int64) {
	// flush out next channel first
	select {
	case <-s.nextchan:
	default:
	}

	s.nextchan <- val
}

// getFirstZeroBit return the first zero bit
// if x[startbit] == 0, return startbit
// if no zero, return startbit
func getFirstZeroBit(x byte, startbit uint) uint {
	for i := uint(0); i < 8; i++ {
		var mask byte = 1 << ((startbit + i) % 8)
		if x&mask == 0 {
			return (startbit + i) % 8
		}
	}
	return startbit
}

// get next non 0xFF byte
func getNextNonFFByte(circle []byte, start_byte, start_bit uint) uint {
	ln := uint(len(circle))
	var i = start_byte
	if circle[i]>>start_bit == 0xff>>start_bit {
		i++
	} else {
		return i
	}
	for ; circle[i%ln] == 0xFF; i++ {
		if i-ln == start_byte {
			return start_byte
		}
	}
	return i % ln
}

func getNextMissingIndex(circle []byte, start_value int64, start_byte, start_bit uint) (int64, uint, uint) {
	ln := uint(len(circle))

	byt := getNextNonFFByte(circle, start_byte, start_bit)
	sbit := start_bit
	if byt != start_byte {
		sbit = 0
	}
	bit := getFirstZeroBit(circle[byt], sbit)

	if bit == 0 { // got 0 -> decrease 1 byte
		byt = (byt + ln - 1) % ln
		bit = 8
	}
	bit--

	if byt < start_byte {
		byt += ln
	}

	dist := (int(byt)-int(start_byte))*8 + (int(bit) - int(start_bit))
	if dist < 0 {
		dist += 8
	}
	return start_value + int64(dist), byt % ln, bit
}

func (s *Squasher) Next() <-chan int64 { return s.nextchan }

func (s *Squasher) GetStatus() string {
	ofs, _, _ := getNextMissingIndex(s.circle, s.start_value, s.start_byte, s.start_bit)
	return fmt.Sprintf("[%d .. %d .. %d]", s.start_value, ofs, s.latest)
}
