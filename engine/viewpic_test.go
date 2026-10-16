package engine

import "testing"

func TestPicFile(t *testing.T) {
	for _, c := range []struct {
		name  string
		block byte
		want  string
	}{{"PIC", 60, "4"}, {"PIC", 27, "5"}, {"PIC", 2, "3"}, {"SPRIT", 9, "1"}, {"FINAL", 60, "3"}} {
		if got := picFile(c.name, c.block, "3"); got != c.want {
			t.Errorf("picFile(%s, %d) = %s, want %s", c.name, c.block, got, c.want)
		}
	}
}

func TestCommas(t *testing.T) {
	for n, want := range map[int]string{7: "7", 202700: "202,700", 1234567: "1,234,567", 1000: "1,000"} {
		if got := commas(n); got != want {
			t.Errorf("commas(%d) = %s", n, got)
		}
	}
}
