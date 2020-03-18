// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

// +build !windows

package keepalive

import (
	"flag"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/felixge/tcpkeepalive"
)

var (
	// DefaultKeepAliveIdleTime specifies how long connection can be idle
	// before sending keepalive message.
	DefaultKeepAliveIdleTime = 15 * time.Minute
	// DefaultKeepAliveCount specifies maximal number of keepalive messages
	// sent before marking connection as dead.
	DefaultKeepAliveCount = 8
	// DefaultKeepAliveInterval specifies how often retry sending keepalive
	// messages when no response is received.
	DefaultKeepAliveInterval = 5 * time.Second
)

// Configuration used to setup the keepalive
type KeepAlive struct {
	KeepAliveIdleTime time.Duration
	KeepAliveCount    int
	KeepAliveInterval time.Duration
}

func (k *KeepAlive) String() string {
	return fmt.Sprintf("KeepAlive { idle_time: %v, count: %d, interval: %v }", (*k).KeepAliveIdleTime, (*k).KeepAliveCount, (*k).KeepAliveInterval)
}

func (k *KeepAlive) Set(conn net.Conn) error {
	return tcpkeepalive.SetKeepAlive(conn, (*k).KeepAliveIdleTime, (*k).KeepAliveCount, (*k).KeepAliveInterval)
}

func NewKeepAlive(idleTime string, count int, interval string) (*KeepAlive, error) {
	idle, err := time.ParseDuration(idleTime)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KeepAliveIdleTime [%s], [%v]", idleTime, err)
	}

	_interval, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("failed to parse KeepAliveIdleInterval [%s], [%v]", interval, err)
	}

	return &KeepAlive{
		KeepAliveIdleTime: idle,
		KeepAliveCount:    count,
		KeepAliveInterval: _interval,
	}, nil
}

type Config struct {
	KeepAliveIdleTime string `yaml:"idle_time"`
	KeepAliveCount    int    `yaml:"count"`
	KeepAliveInterval string `yaml:"interval"`
}

func (c *Config) String() string {
	return fmt.Sprintf("idle_time=%v, count=%d, interval=%v", (*c).KeepAliveIdleTime, (*c).KeepAliveCount, (*c).KeepAliveInterval)
}

func (c *Config) Set(opt string) error {
	opts := strings.Split(opt, ",")

	for _, opt := range opts {
		keyVal := strings.Split(opt, "=")
		if len(keyVal) != 2 {
			return fmt.Errorf("failed to parse keepalive config from option: %s. " +
				"Options must be formatted as \"key=value\"", opt)
		}

		key := strings.TrimSpace(keyVal[0])
		val := strings.TrimSpace(keyVal[1])

		if len(val) == 0 {
			return fmt.Errorf("failed to parse keepalive config from option: %s. " +
				"Option values cannot be empty", opt)
		}

		if key == "idle_time" {
			c.KeepAliveIdleTime = val
		} else if key == "count" {
			count, err := strconv.ParseInt(val, 10, 0)
			if err != nil {
				return fmt.Errorf("failed to parse \"count\" from: %s", val)
			}
			c.KeepAliveCount = int(count)
		} else if key == "interval" {
			c.KeepAliveInterval = val
		} else {
			return fmt.Errorf("unknow keep-alive option \"%s\"", keyVal[0])
		}
	}

	return nil
}

func (c *Config) Parse() (*KeepAlive, error) {
	return NewKeepAlive(c.KeepAliveIdleTime, c.KeepAliveCount, c.KeepAliveInterval)
}

func NewDefaultConfig() *Config {
	return &Config{
		KeepAliveIdleTime: DefaultKeepAliveIdleTime.String(),
		KeepAliveCount:    DefaultKeepAliveCount,
		KeepAliveInterval: DefaultKeepAliveInterval.String(),
	}
}

func AddKeepAliveFlag() *Config {
	fmtMsg := "Format for \"idle_time\" and \"interval\" values are value[unit], where unit can be one of \"ns\", \"us\" (or \"µs\"), \"ms\", \"s\", \"m\" or \"h\".\nFor example, 1 second == 1s or 10 minutes 5 seconds == 10m5s."

	config := NewDefaultConfig()

	flag.Var(config, "keep-alive", "TCP keep alive configuration. " +
		"Use coma separated key/value pairs [key=value,key2=value2...]. Keys are: idle_time, count and interval.\n"+fmtMsg+
		"\nThe value for count is an integer\nExample: --keep-alive idle_time=1m, count=5, interval=5s")
	return config
}

func SetDefaultKeepAlive(conn net.Conn) error {
	return tcpkeepalive.SetKeepAlive(conn, DefaultKeepAliveIdleTime, DefaultKeepAliveCount, DefaultKeepAliveInterval)
}
