// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

package keepalive

import (
	"testing"
	"time"
)

func TestSetConfig(t *testing.T) {

	tests := []struct {
		options  string
		expected Config
	}{
		{
			options: "idle_time=10m, count=10, interval=10s",
			expected: Config{
				KeepAliveIdleTime: "10m",
				KeepAliveCount:    10,
				KeepAliveInterval: "10s",
			},
		},
		{
			options: "idle_time=10m,count=10,interval=10s",
			expected: Config{
				KeepAliveIdleTime: "10m",
				KeepAliveCount:    10,
				KeepAliveInterval: "10s",
			},
		},
		{
			options: "idle_time=10m,           count=10, interval=10s",
			expected: Config{
				KeepAliveIdleTime: "10m",
				KeepAliveCount:    10,
				KeepAliveInterval: "10s",
			},
		},
		{
			options: "idle_time=10m,interval=10s",
			expected: Config{
				KeepAliveIdleTime: "10m",
				KeepAliveCount:    DefaultKeepAliveCount,
				KeepAliveInterval: "10s",
			},
		},
		{
			options: "interval=10s,idle_time=10m",
			expected: Config{
				KeepAliveIdleTime: "10m",
				KeepAliveCount:    DefaultKeepAliveCount,
				KeepAliveInterval: "10s",
			},
		},
	}

	for _, test := range tests {
		config := NewDefaultConfig()
		err := config.Set(test.options)
		if err != nil {
			t.Errorf("Error not expected, but got error: %s", err)
		}
		if config.KeepAliveIdleTime != test.expected.KeepAliveIdleTime {
			t.Errorf("Expected KeepAliveIdleTime to be \"%s\" but was \"%s\"", test.expected.KeepAliveInterval, config.KeepAliveIdleTime)
		}

		if config.KeepAliveCount != test.expected.KeepAliveCount {
			t.Errorf("Expected KeepAliveCount to be \"%d\" but was \"%d\"", test.expected.KeepAliveCount, config.KeepAliveCount)
		}

		if config.KeepAliveInterval != test.expected.KeepAliveInterval {
			t.Errorf("Expected KeepAliveInterval to be \"%s\" but was \"%s\"", test.expected.KeepAliveInterval, config.KeepAliveInterval)
		}
	}
}

func TestSetConfigErrors(t *testing.T) {
	config := NewDefaultConfig()

	options := []string{
		"idle_time",
		"count=a",
		"idle_time=",
		"balony_prop",
	}

	for _, opt := range options {
		err := config.Set(opt)

		if err == nil {
			t.Errorf("Expected and error but got none")
		}
	}

}

func TestParseConfig(t *testing.T) {
	tests := []struct {
		options  string
		expected KeepAlive
	}{
		{
			options: "idle_time=10m,count=10,interval=10s",
			expected: KeepAlive{
				KeepAliveIdleTime: 10 * time.Minute,
				KeepAliveCount:    10,
				KeepAliveInterval: 10 * time.Second,
			},
		},
		{
			options: "idle_time=10h,count=10,interval=10s",
			expected: KeepAlive{
				KeepAliveIdleTime: 10 * time.Hour,
				KeepAliveCount:    10,
				KeepAliveInterval: 10 * time.Second,
			},
		},
	}

	for _, test := range tests {
		config := NewDefaultConfig()
		err := config.Set(test.options)
		keepAlive, err := config.Parse()
		if err != nil {
			t.Errorf("Error not expected, but got error: %s", err)
		}
		if keepAlive.KeepAliveIdleTime != test.expected.KeepAliveIdleTime {
			t.Errorf("Expected KeepAliveIdleTime to be \"%v\" but was \"%v\"", test.expected.KeepAliveInterval, keepAlive.KeepAliveIdleTime)
		}

		if keepAlive.KeepAliveCount != test.expected.KeepAliveCount {
			t.Errorf("Expected KeepAliveCount to be \"%d\" but was \"%d\"", test.expected.KeepAliveCount, keepAlive.KeepAliveCount)
		}

		if keepAlive.KeepAliveInterval != test.expected.KeepAliveInterval {
			t.Errorf("Expected KeepAliveInterval to be \"%v\" but was \"%v\"", test.expected.KeepAliveInterval, keepAlive.KeepAliveInterval)
		}
	}
}

func TestSeParseConfigErrors(t *testing.T) {
	config := NewDefaultConfig()

	options := []string{
		"idle_time=10", // no unit
		"idle_time=10l", // not a known unit
		"interval=10", // no unit
		"interval=10l", // not a known unit
	}

	for _, opt := range options {
		err := config.Set(opt)
		if err != nil {
			t.Errorf("Expected and error but got none")
		}
		_, err = config.Parse()
		if err == nil {
			t.Errorf("Expected and error but got none")
		}
	}
}
