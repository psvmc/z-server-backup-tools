package util

import "testing"

func TestFormatDuration(t *testing.T) {
	cases := map[int64]string{
		0:     "0:00",
		59000: "0:59",
		60000: "1:00",
		3661000: "1:01:01",
	}
	for ms, want := range cases {
		if got := FormatDuration(ms); got != want {
			t.Fatalf("FormatDuration(%d) = %q, want %q", ms, got, want)
		}
	}
}
