package squasher

import (
	"testing"
	"time"
)

func TestSquasherLimit(t *testing.T) {
	//t.Skip()
	sq := NewSquasher(0, 1000)

	go func() {
		for i := range sq.Next() {
			if i == 1000 {
				break
			}
		}
	}()

	for i := 0; i < 1000; i++ {
		sq.Mark(int64(i))
	}

	err := sq.Mark(20000)
	if err == nil {
		t.Errorf("should be error, got nil")
	}
}

func TestFirstZeroBit(t *testing.T) {
	//t.Skip()
	cs := []struct {
		x   byte
		out uint
	}{
		{0xFF, 8},
		{0xF1, 1}, // 1111 0001
	}

	for _, c := range cs {
		out := getFirstZeroBit(c.x)
		if out != c.out {
			t.Errorf("with byte %x, expect %d, got %d", c.x, c.out, out)
		}
	}
}

func TestNextMissingIndex(t *testing.T) {
	//t.Skip()
	cs := []struct {
		circle            []byte
		value             int64
		byt, bit          uint
		nextvalue         int64
		nextbyte, nextbit uint
	}{
		{[]byte{0x0F, 0, 0}, 2100, 0, 1, 2102, 0, 3},    // - 1111 | - - | - -
		{[]byte{0xFF, 0x1F, 0}, 2100, 0, 2, 2110, 1, 4}, // 1111 1111 | 0001 1111 | --
		{[]byte{0, 0xFF, 0x05}, 2100, 1, 4, 2104, 2, 0}, // -- | 1111 1111 | - 0101
		{[]byte{0, 0xFF, 0}, 2100, 1, 4, 2103, 1, 7},    // -- | 1111 1111 | -
		{[]byte{0x0F, 0, 0xFF}, 2100, 2, 0, 2111, 0, 3}, // - 1111 | - | 1111 1111
		{[]byte{0xff, 0xff, 0xff, 0x3f, 0}, 0, 0, 0, 29, 3, 5},
		{[]byte{0xff, 0, 0xff, 0xff}, 0, 2, 0, 23, 0, 7},
		{[]byte{0xff, 0xff, 0xff, 0}, 0, 1, 0, 15, 2, 7},
		{[]byte{0, 0xff}, 0, 1, 0, 7, 1, 7},
	}

	for i, c := range cs {
		nval, nbyt, nbit := getNextMissingIndex(c.circle, c.value, c.byt, c.bit)
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
	//t.Skip()
	ts := []struct {
		circle             []byte
		start_byte, output uint
	}{
		{[]byte{0xFF, 0xF4, 0}, 0, 1},
		{[]byte{0xFF, 0xFF, 0xF4, 0xFF, 0xFE, 0}, 3, 4},
		{[]byte{0xFE, 0xFF, 0}, 0, 0},
		{[]byte{0xFF, 0xFF, 0}, 0, 2},
		{[]byte{0, 0xFF, 0xFF}, 1, 0},
	}
	for _, c := range ts {
		out := getNextNonFFByte(c.circle, c.start_byte)
		if out != c.output {
			t.Errorf("with circle %v and start_byte %d, expect %d, got %d", c.circle, c.start_byte, c.output, out)
		}
	}
}

func TestSquasherClean(t *testing.T) {
	sq := NewSquasher(0, 23)

	errc := make(chan int64)
	go func() {
		for i := range sq.Next() {
			if i > 30 {
				errc <- i
				break
			}
		}
	}()


	for i := 1; i < 23; i++ {
		sq.Mark(int64(i))
	}

	sq.Mark(0)
	sq.Mark(23); sq.Mark(24); sq.Mark(25); sq.Mark(26)
	sq.Mark(27); sq.Mark(28); sq.Mark(29); sq.Mark(30)
	select {
	case i := <-errc:
		t.Errorf("got 30 before commit: %d", i)
	case <-time.After(2 * time.Second):
	}
}

func TestSquasher(t *testing.T) {
	//t.Skip()
	sq := NewSquasher(0, 4)

	gotc := make(chan int64)

	go func() {
		for i := range sq.Next() {
			gotc <- i
		}
	}()


	sq.Mark(2)
	sq.Mark(3)
	sq.Mark(4)
	sq.Mark(1)

	select {
	case <-gotc:
		t.Error("should not call this")
	default:

	}

	sq.Mark(0)
	out := <-gotc
	if out != 4 {
		t.Errorf("expect 4, got %d", out)
	}
}

func TestSquasher100(t *testing.T) {
	//t.Skip()
	sq := NewSquasher(0, 500)

	gotc := make(chan int64)

	go func() {
		for i := range sq.Next() {
			gotc <- i
		}
	}()
	for i := int64(1); i <= 400; i++ {
		sq.Mark(i)

	}
	select {
	case <-gotc:
		t.Error("should not call this")
	default:

	}
	sq.Mark(0)
	out := <-gotc
	if out != 400 {
		t.Errorf("expect 400, got %d", out)
	}
}

func TestSquasherTurnAround(t *testing.T) {
	//t.Skip()
	sq := NewSquasher(0, 100)

	donec := make(chan bool, 0)
	go func() {
		for i := range sq.Next() {
			if i == 1000 {
				break
			}
		}
		donec <- true
	}()

	for i := 0; i <= 1000; i++ {
		sq.Mark(int64(i))
	}

	<-donec
}

func TestSetBit(t *testing.T) {
	//t.Skip()
	ts := []struct {
		circle             []byte
		start_byte, start_bit uint
		start_value, val int64
		out_circle []byte
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
