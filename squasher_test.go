package squasher

import (
	"testing"
)

func TestZeroCirle(t *testing.T) {
	ts := []struct {
		circle                           []byte
		frombyte, frombit, tobyte, tobit uint
		out_circle                       []byte
	}{
		// don't need to expand
		{[]byte{0b11111111}, 0, 1, 0, 3, []byte{0b11111001}},
		{[]byte{0b11111111}, 0, 1, 0, 1, []byte{0b11111111}},
		{[]byte{0b11111111}, 0, 4, 0, 3, []byte{0b00001000}},
		{[]byte{0b11111111}, 0, 4, 0, 7, []byte{0b10001111}},
		{[]byte{0b11111111}, 0, 4, 0, 9, []byte{0b10001111}}, // 9 is considered as 7

		{[]byte{0b11111111, 0b11111111}, 0, 4, 1, 2, []byte{0b00001111, 0b11111100}},
		{[]byte{0b11111111, 0b11111111}, 1, 2, 0, 4, []byte{0b11110000, 0b00000011}},
		{[]byte{0b11111111, 0b11111111}, 0, 4, 0, 1, []byte{0b00001110, 0b00000000}},
		{[]byte{0b11111111, 0b11111111}, 0, 1, 0, 4, []byte{0b11110001, 0b11111111}},
	}
	for it, c := range ts {
		zeroCircle(c.circle, c.frombyte, c.tobyte, c.frombit, c.tobit)
		if len(c.out_circle) != len(c.circle) {
			t.Errorf("should be equal, expect %d, got %d",
				len(c.out_circle), len(c.circle))
		}
		for i := range c.out_circle {
			if c.out_circle[i] != c.circle[i] {
				t.Errorf("expect %08b, got %08b at index %d, of test %d", c.out_circle[i], c.circle[i], i, it)
			}
		}
	}
}

func TestFirstZeroBit(t *testing.T) {
	cs := []struct {
		x     byte
		start uint
		out   uint
	}{
		{0xFF, 0, 0}, // 1111 1111
		{0xF1, 0, 1}, // 1111 0001
		{0x06, 1, 3}, // 0000 0110
		{0x06, 2, 3}, // 0000 0110
		{0x06, 3, 3}, // 0000 0110
		{0xE7, 5, 3}, // 1110 0111
	}

	for _, c := range cs {
		out := getFirstZeroBit(c.x, c.start)
		if out != c.out {
			t.Errorf("with byte %x, expect %d, got %d", c.x, c.out, out)
		}
	}
}

func TestNextStart(t *testing.T) {
	cs := []struct {
		circle            []byte
		value             int64
		byt, bit          uint
		nextvalue         int64
		nextbyte, nextbit uint
	}{
		{[]byte{0b00001110, 0, 0}, 2100, 0, 1, 2102, 0, 3},
		{[]byte{0b11111100, 0b00011111, 0}, 2100, 0, 2, 2110, 1, 4},
		{[]byte{0, 0b11110000, 0b00000101}, 2100, 1, 4, 2104, 2, 0},
		{[]byte{0, 0b11110000, 0}, 2100, 1, 4, 2103, 1, 7},
		{[]byte{0b00001111, 0, 0b11111111}, 2100, 2, 0, 2111, 0, 3},
		{[]byte{0xff, 0xff, 0xff, 0x3f, 0}, 0, 0, 0, 29, 3, 5},
		{[]byte{0xff, 0, 0xff, 0xff}, 0, 2, 0, 23, 0, 7},
		{[]byte{0xff, 0xff, 0xff, 0}, 0, 1, 0, 15, 2, 7},
		{[]byte{0, 0xff}, 0, 1, 0, 7, 1, 7},
		{[]byte{0x81}, 2064, 0, 7, 2065, 0, 0},
	}

	for i, c := range cs {
		nval, nbyt, nbit := getNextStart(c.circle, c.value, c.byt, c.bit)
		if nval != c.nextvalue {
			t.Errorf("test %d, expect val %d, got %d", i, c.nextvalue, nval)
		}

		if nbyt != c.nextbyte {
			t.Errorf("test %d, expect val %d, got %d", i, c.nextbyte, nbyt)
		}

		if nbit != c.nextbit {
			t.Errorf("test %d, expect val %d, got %d", i, c.nextbit, nbit)
		}
	}
}

func TestNextNonFFByte(t *testing.T) {
	ts := []struct {
		circle                        []byte
		start_byte, start_bit, output uint
	}{
		{[]byte{0xFF, 0xF4, 0}, 0, 0, 1},
		{[]byte{0xFF, 0xFF, 0xF4, 0xFF, 0xFE, 0}, 3, 0, 4},
		{[]byte{0xFE, 0xFF, 0}, 0, 0, 0},
		{[]byte{0xFF, 0xFF, 0}, 0, 0, 2},
		{[]byte{0, 0xFF, 0xFF}, 1, 0, 0},
		{[]byte{0, 0xF0, 0xFF}, 1, 4, 0},
		{[]byte{0, 0xF0, 0xFF}, 1, 0, 1},
	}
	for _, c := range ts {
		out := getNextNonFFByte(c.circle, c.start_byte, c.start_bit)
		if out != c.output {
			t.Errorf("with circle %v and start_byte %d, expect %d, got %d", c.circle, c.start_byte, c.output, out)
		}
	}
}

func TestSquasher(t *testing.T) {
	sq := NewSquasher(0)

	for i := 1; i <= 2300; i++ {
		last := sq.Mark(int64(i))
		if last != -1 {
			t.Errorf("should equal -1, got %d", last)
		}
	}

	last := sq.Mark(0)
	if last != 2300 {
		t.Errorf("should equal 2300, got %d", last)
	}
}

func TestSquasherDupMark(t *testing.T) {
	sq := NewSquasher(0)

	for i := 1; i <= 23; i++ {
		sq.Mark(int64(i))
	}

	last := sq.Mark(0)
	if last != 23 {
		t.Errorf("should equal 22, got %d", last)
	}

	for i := 1; i <= 10; i++ {
		if last := sq.Mark(int64(i)); last != 23 {
			t.Errorf("should equal 22, got %d", last)
		}
	}
}

func TestSquasherTurnAround2(t *testing.T) {
	sq := NewSquasher(0)
	cursize := len(sq.circle)
	last := int64(0)
	for i := 0; i <= 10000; i++ {
		last = sq.Mark(int64(i))
	}

	if last != 10000 {
		t.Errorf("should equal, got %d", last)
	}

	if len(sq.circle) != cursize {
		t.Errorf("should be %d, got %d", cursize, len(sq.circle))
	}
}

func TestSetBit(t *testing.T) {
	ts := []struct {
		circle                []byte
		start_byte, start_bit uint
		start_value, val      int64
		out_circle            []byte
	}{
		{[]byte{0, 0xF5, 0xD1, 0xF6}, 1, 0, 10, 20, []byte{0, 0xF5, 0xD5, 0xF6}},
		{[]byte{0, 0x80, 0xD5, 0xB4}, 1, 7, 10, 20, []byte{0, 0x80, 0xD5, 0xB6}},
	}
	for it, c := range ts {
		setBit(c.circle, c.start_byte, c.start_bit, c.start_value,
			c.val)
		if len(c.out_circle) != len(c.circle) {
			t.Errorf("should be equal, expect %d, got %d",
				len(c.circle), len(c.out_circle))
		}
		for i := range c.out_circle {
			if c.out_circle[i] != c.circle[i] {
				t.Errorf("expect %d, got %d at index %d, of test %d", c.out_circle[i], c.circle[i], i, it)
			}
		}
	}
}

func TestExpandCircle(t *testing.T) {
	ts := []struct {
		circle                []byte
		start_byte, start_bit uint
		newsize               uint
		out_circle            []byte
	}{
		// don't need to expand
		{[]byte{0b11110001, 0b11111111}, 0, 4, 10, []byte{0b11110001, 0b11111111}},

		// expand to 4 bytes
		{[]byte{0b11110001, 0b11111111}, 0, 4, 20, []byte{0b11110000, 0b11111111, 0b00000001, 0b00000000}},

		// expand to 8 bytes
		{[]byte{0b11111111, 0b10001111}, 1, 7, 62, []byte{0b10000000, 0b11111111, 0b00001111, 0b00000000, 0b00000000, 0b00000000, 0b00000000, 0b00000000}},
	}
	for it, c := range ts {
		newcircle := expandCircle(c.circle, c.start_byte, c.start_bit, c.newsize)
		if len(c.out_circle) != len(newcircle) {
			t.Errorf("should be equal, expect %d, got %d",
				len(c.out_circle), len(newcircle))
		}
		for i := range c.out_circle {
			if c.out_circle[i] != newcircle[i] {
				t.Errorf("expect %d, got %d at index %d, of test %d", c.out_circle[i], newcircle[i], i, it)
			}
		}
	}
}
