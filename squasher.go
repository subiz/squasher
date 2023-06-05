package squasher

import (
	"fmt"
	"sync"
)

type Squasher struct {
	*sync.Mutex
	circle []byte

	start_value int64
	start_byte  uint
	start_bit   uint

	latest int64
}

func (s *Squasher) Print() {
	fmt.Printf("CIRCLE(%d) %d:%d=%d ------\n", len(s.circle), s.start_byte, s.start_bit, s.start_value)
	for _, c := range s.circle {
		fmt.Printf("%08b ", c)
	}
	fmt.Println("")
}

// a circle with size = 2
//
//	start_byte: 1 ---v     v-- start_bit: 4
//
// byt 0 0 0 0 0 0 0 0  1 1 1 1 1 1 1 1  2 2 2 2 2 2 2 2  3 3 3 3 3 3 3 3
// bit 7 6 5 4 3 2 1 0  7 6 5 4 3 2 1 0  7 6 5 4 3 2 1 0  7 6 5 4 3 2 1 0
// cir 0 0 0 0 0 0 0 0  0 0 0 1 0 0 1 0  1 0 0 0 1 1 0 1  1 1 1 1 0 1 0 0
func NewSquasher() *Squasher {
	return &Squasher{Mutex: &sync.Mutex{}}
}

// zeroCircle zero all bits from byte to byte
func zeroCircle(circle []byte, frombyt, tobyt, frombit, tobit uint) {
	ln := uint(len(circle))
	if ln == 0 {
		return
	}

	// zero out first byte
	var mask byte

	// handle the first end the last byte
	if frombyt == tobyt || ln == 1 {
		if frombit > 7 {
			return // invalid
		}

		if tobit > 7 {
			tobit = 7
		}

		if frombit == tobit {
			return // do nothing
		}

		if frombit < tobit {
			for i := frombit; i < tobit; i++ {
				mask |= 1 << (i % 8)
			}
			mask ^= 0xFF
			circle[frombyt] &= mask
			return
		}

		for i := frombit; i%8 != tobit; i++ {
			mask |= 1 << (i % 8)
		}
		mask ^= 0xFF
		circle[frombyt] &= mask

		// set other byte to zero
		for i := uint(0); i < ln; i++ {
			if i != frombyt {
				circle[i] = 0
			}
		}
		return
	}

	circle[frombyt] = circle[frombyt] << (8 - frombit) >> (8 - frombit)
	circle[tobyt] = circle[tobyt] >> tobit << tobit
	for i := frombyt + 1; i%ln != tobyt; i++ {
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

// return new cirle, keep the start bit
func expandCircle(circle []byte, start_byte, start_bit, newsize uint) []byte {
	newlen := uint(len(circle)) // in byte
	if newlen*8 > newsize+1 {
		// still enough room, don't alloc new size
		return circle
	}

	for {
		newlen = newlen * 2
		if newlen*8 > newsize+1 {
			break
		}
	}
	newcircle := make([]byte, newlen)

	oldlen := uint(len(circle))
	for i := uint(0); i < oldlen; i++ {
		newcircle[i] = circle[(start_byte+i)%oldlen]
	}

	// close the circle
	newcircle[oldlen] = circle[start_byte] << (8 - start_bit) >> (8 - start_bit)
	newcircle[0] = circle[start_byte] >> start_bit << start_bit
	return newcircle
}

func (s *Squasher) Init(i int64) {
	s.Lock()
	defer s.Unlock()

	if s.circle != nil {
		return
	}
	// init
	circle := make([]byte, 32)
	circle[0] = 1
	s.start_value = i - 1
	s.start_byte = 0
	s.start_bit = 0
	s.circle = circle
}

// Mark a value i as processed
func (s *Squasher) Mark(i int64) int64 {
	s.Lock()
	defer s.Unlock()

	if s.circle == nil {
		// init
		circle := make([]byte, 32)
		circle[0] = 1
		s.start_value = i - 1
		s.start_byte = 0
		s.start_bit = 0
		s.circle = circle
	}

	if i > s.latest {
		s.latest = i
	}
	dist := i - s.start_value
	if dist <= 0 {
		return s.start_value
	}
	ln := uint(len(s.circle))
	if uint(dist)+1 > ln*8 {
		s.circle = expandCircle(s.circle, s.start_byte, s.start_bit, uint(dist))
		s.start_byte = 0
	}

	setBit(s.circle, s.start_byte, s.start_bit, s.start_value, i)

	if dist != 1 {
		return s.start_value
	}
	nextval, nextbyte, nextbit :=
		getNextStart(s.circle, s.start_value, s.start_byte, s.start_bit)

	zeroCircle(s.circle, s.start_byte, nextbyte, s.start_bit, nextbit)
	s.start_value, s.start_byte, s.start_bit = nextval, nextbyte, nextbit
	return s.start_value
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

func getNextStart(circle []byte, start_value int64, start_byte, start_bit uint) (int64, uint, uint) {
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

func (s *Squasher) GetStatus() string {
	ofs, _, _ := getNextStart(s.circle, s.start_value, s.start_byte, s.start_bit)
	return fmt.Sprintf("[%d .. %d .. %d]", s.start_value, ofs, s.latest)
}
