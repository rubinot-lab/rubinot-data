package scraper

import "testing"

func TestVocationNameByID(t *testing.T) {
	tests := []struct {
		name     string
		id       int
		expected string
	}{
		{name: "known base vocation", id: 2, expected: "Druid"},
		{name: "known exalted monk vocation", id: 10, expected: "Exalted Monk"},
		{name: "unknown vocation", id: 999, expected: ""},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := vocationNameByID(tt.id)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
