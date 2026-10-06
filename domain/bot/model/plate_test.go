package model

import "testing"

func TestParsePlate(t *testing.T) {
	tests := []struct {
		text  string
		plate string
		ok    bool
	}{
		{text: "а000аа78", plate: "а000аа78", ok: true},
		{text: " а000аа178, ", plate: "а000аа178", ok: true},
		{text: "a000aa78.", plate: "a000aa78", ok: true},
		{text: "а000аа7", ok: false},
		{text: "000аа78", ok: false},
		{text: "+70000000000", ok: false},
	}
	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			plate, ok := ParsePlate(tt.text)

			if plate != tt.plate || ok != tt.ok {
				t.Errorf("got (%q, %v), want (%q, %v)", plate, ok, tt.plate, tt.ok)
			}
		})
	}
}
