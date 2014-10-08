package lib

import (
	"testing"
)

func TestPretty(t *testing.T) {
	for k, v := range map[int64]string{
		-1:         "-1",
		0:          "0",
		999:        "999",
		1000:       "1,000",
		1234567890: "1,234,567,890",
	} {
		if got, want := Pretty(k), v; got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	}
}
