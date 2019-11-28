package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"
)

type Stats struct {
	Attempt   int
	Interval  time.Duration
	Config    *RetryConfig
	breakNext bool
}

type RetryConfig struct {
	Maximum         int           `yaml:"maximum" hcl:"maximum"`
	Interval        time.Duration `yaml:"interval" hcl:"-"`
	IntervalSeconds int           `yaml:"-" hcl:"interval_seconds"`
	Increment       bool          `yaml:"increment" hcl:"increment"`
	IntervalMaximum time.Duration `yaml:"intervalMaximum" hcl:"interval_maximum"`
	Forever         bool          `yaml:"forever" hcl:"forever"`
	Jitter          bool          `yaml:"jitter" hcl:"jitter"`
}

func (s *Stats) String() string {
	str := fmt.Sprintf("Attempt %d/", s.Attempt)
	if s.Config.Forever {
		str = str + "∞"
	} else {
		str = str + fmt.Sprintf("%d", s.Config.Maximum)
	}

	if s.Config.Interval > 0 {
		str = str + fmt.Sprintf(" Retrying in %s", s.Interval)
	}

	return str
}

func (s *Stats) Break() {
	s.breakNext = true
}

func Retry(callback func(*Stats) error, config *RetryConfig) error {
	var err error
	if config == nil {
		config = &RetryConfig{
			Forever:  true,
			Interval: 1 * time.Second,
			Jitter:   false,
		}
	}

	if config.Maximum == 0 && !config.Forever {
		config.Maximum = 10
	}

	if config.Increment && config.IntervalMaximum == 0 {
		config.IntervalMaximum = time.Minute
	}

	if config.IntervalSeconds > 0 {
		config.Interval = time.Duration(config.IntervalSeconds) * time.Second
	}

	if config.Forever && config.Interval == 0 {
		return errors.New("You can't do a forever retry with no interval")
	}

	stats := &Stats{Attempt: 1, Config: config}
	random := rand.New(rand.NewSource(time.Now().UnixNano()))

	for {
		if config.Increment {
			stats.Interval = stats.Interval + config.Interval
		} else {
			stats.Interval = config.Interval
		}
		if config.IntervalMaximum > 0 && stats.Interval > config.IntervalMaximum {
			stats.Interval = config.IntervalMaximum
		}
		if config.Jitter {
			stats.Interval = stats.Interval + (time.Duration(1000*random.Float32()) * time.Millisecond)
		}
		err = callback(stats)
		if err == nil {
			return nil
		}
		if stats.breakNext {
			return err
		}
		stats.Attempt = stats.Attempt + 1
		time.Sleep(stats.Interval)
		if !stats.Config.Forever {
			if stats.Attempt > stats.Config.Maximum {
				break
			}
		}
	}

	return err
}
