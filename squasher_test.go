package squasher

import (
	"testing"
)

func TestSquasherLimit(t *testing.T) {
	t.Skip()
	sq := NewSquasher(0, 100)

	for i := 0; i < 1000; i++ {
		sq.Mark(int64(i))
	}

	for i := range sq.Next() {
		if i == 1000 {
			break
		}
	}

	err := sq.Mark(2000)
	if err == nil {
		t.Errorf("should be error, got nil")
	}

}

func TestFirstZeroBit(t *testing.T) {
	cs := []struct {
		x byte
		out uint
	} {
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
	cs := []struct {
		circle []byte
		value int64
		byt, bit uint
		nextvalue int64
		nextbyte, nextbit uint
	} {
		{nil, 0, 0, 0, 0, 0, 0},
		{nil, 1, 4, 2, 0, 0, 0},
		{[]byte{0x0F, 0, 0}, 2100, 0, 1, 2102, 0, 3}, // - 1111 | - - | - -
		{[]byte{0xFF, 0x1F, 0}, 2100, 0, 2, 2110, 1, 4}, // 1111 1111 | 0001 1111 | --
		{[]byte{0, 0xFF, 0x05}, 2100, 1, 4, 2104, 2, 0}, // -- | 1111 1111 | - 0101
		{[]byte{0, 0xFF, 0}, 2100, 1, 4, 2103, 1, 7}, // -- | 1111 1111 | -
		{[]byte{0x0F, 0, 0xFF}, 2100, 2, 0, 2111, 0, 3}, // - 1111 | - | 1111 1111
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
	ts := []struct {
		circle             []byte
		start_byte, output uint
	} {
		{[]byte{0xFF, 0xF4},0, 1},
		{[]byte{0xFF, 0xFF, 0xF4, 0xFF, 0xFE}, 3, 4},
		{[]byte{},0, 0},
		{[]byte{0xFE, 0xFF}, 0, 0},
		{[]byte{0xFF, 0xFF}, 0, 0},
		{[]byte{0xFF, 0xFF}, 10, 0},
	}
	for _, c := range ts {
		out := getNextNonFFByte(c.circle, c.start_byte)
		if out != c.output {
			t.Errorf("with circle %v and start_byte %d, expect %d, got %d", c.circle, c.start_byte, c.output, out)
		}
	}
}

func TestSquasher(t *testing.T) {
	t.Skip()
	sq := NewSquasher(0, 1000)

	gotc := make(chan int64)

	go func() {
		for i := range sq.Next() {
			gotc <- i
		}
	}()

	sq.Mark(1)
	sq.Mark(2)
	sq.Mark(3)
	sq.Mark(4)

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
