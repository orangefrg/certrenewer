package filehelper

import (
	"testing"
	"time"
)

func TestStringToTime(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		hasError bool
	}{
		{"1d", 24 * time.Hour, false},
		{"2h", 2 * time.Hour, false},
		{"30m", 30 * time.Minute, false},
		{"45s", 45 * time.Second, false},
		{"500ms", 500 * time.Millisecond, false},
		{"100us", 100 * time.Microsecond, false},
		{"200ns", 200 * time.Nanosecond, false},
		{"1.5h", 90 * time.Minute, false},
		{"2.5d", 60 * time.Hour, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"1d2h30m15s500ms100ns",
			24*time.Hour + 2*time.Hour + 30*time.Minute +
				15*time.Second + 500*time.Millisecond + 100*time.Nanosecond, false},
		{"invalid", 0, true},
		{"10x", 0, true},
	}

	for _, test := range tests {
		result, err := StringToTime(test.input)
		if (err != nil) != test.hasError {
			t.Errorf("StringToTime(%s) error = %v, expected error = %v", test.input, err, test.hasError)
		}
		if result != test.expected {
			t.Errorf("StringToTime(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}
