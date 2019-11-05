package main

import (
	"testing"
	"time"
)

func TestString(t *testing.T) {
	tests := []struct {
		res   string
		stats *Stats
	}{
		{
			res: "Attempt 0/0",
			stats: &Stats{
				Attempt: 0,
				Config: &RetryConfig{
					Forever:  false,
					Interval: 0,
				},
			},
		},
		{
			res: "Attempt 1/∞ Retrying in 1s",
			stats: &Stats{
				Attempt:  1,
				Interval: time.Second,
				Config: &RetryConfig{
					Forever:  true,
					Interval: time.Second,
				},
			},
		},
	}
	for _, test := range tests {
		if test.stats.String() != test.res {
			t.Errorf("Must be %s, but got %s", test.res, test.stats.String())
		}
	}
}
