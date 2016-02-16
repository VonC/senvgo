package inst

import "testing"

func TestCheckInst(t *testing.T) {
	tests := []struct {
		id          string
		isInstalled bool
	}{
		{"prg1", true},
		{"prg2", false},
		{"prg3", true},
	}
	for _, test := range tests {
		b := CheckInst(test.id)
		if b != test.isInstalled {
			t.Errorf("CheckInst '%s': expected '%v', got '%v'", test.id, test.isInstalled, b)
		}
	}
}
