package main

import (
	"errors"
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

func TestRetry(t *testing.T) {
	tests := []*RetryConfig{
		{
			Maximum:  0,
			Forever:  false,
			Interval: 100 * time.Microsecond,
		},
		{
			Increment:       true,
			IntervalSeconds: 1,
		},
		{
			Increment:       true,
			Maximum:         10,
			Interval:        100 * time.Microsecond,
			IntervalMaximum: 50 * time.Microsecond,
			Jitter:          true,
		},
	}
	for _, test := range tests {
		Retry(func(s *Stats) error {
			return nil
		}, test)
	}
	err := Retry(func(s *Stats) error {
		return nil
	}, &RetryConfig{
		Interval: 0,
		Forever:  true,
	})
	if err == nil {
		t.Error("Must be an error, but got nil")
	}
	Retry(func(s *Stats) error {
		if s.Attempt > 0 {
			s.Break()
		}
		return errors.New("Error")
	}, &RetryConfig{
		Interval: 100 * time.Microsecond,
		Forever:  true,
	})
}
