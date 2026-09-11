package mealplan

import (
	"errors"
	"testing"
)

func TestValidateDate(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want error
	}{
		{"valid iso date", "2026-08-24", nil},
		{"slash separator", "2026/08/24", ErrInvalidDate},
		{"no separator", "20260824", ErrInvalidDate},
		{"non-existent month", "2026-13-01", ErrInvalidDate},
		{"empty", "", ErrInvalidDate},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateDate(tc.in)
			if !errors.Is(got, tc.want) {
				t.Errorf("ValidateDate(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidateSlot(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want error
	}{
		{"breakfast", "breakfast", nil},
		{"lunch", "lunch", nil},
		{"dinner", "dinner", nil},
		{"unknown", "snack", ErrInvalidSlot},
		{"empty", "", ErrInvalidSlot},
		{"uppercase", "Dinner", ErrInvalidSlot},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidateSlot(tc.in)
			if !errors.Is(got, tc.want) {
				t.Errorf("ValidateSlot(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestValidatePosition(t *testing.T) {
	cases := []struct {
		name string
		in   int
		want error
	}{
		{"zero", 0, nil},
		{"one", 1, nil},
		{"negative", -1, ErrInvalidPosition},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ValidatePosition(tc.in)
			if !errors.Is(got, tc.want) {
				t.Errorf("ValidatePosition(%d) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
