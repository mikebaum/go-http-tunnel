// Copyright (C) 2017 Michał Matczuk
// Use of this source code is governed by an AGPL-style
// license that can be found in the LICENSE file.

package keepalive

import (
	"fmt"
	"net"
)

var (
	// DefaultKeepAliveEnabled specifies if keep alive is enabled or not.
	DefaultKeepAliveEnabled = true
)

type KeepAlive struct {
	Enabled bool
}

func (k *KeepAlive) Set(conn net.Conn) error {
	return keepAlive(conn, k.Enabled)
}

func (k *KeepAlive) String() string {
	return fmt.Sprintf("KeepAlive { enabled: %v }", (*k).Enabled)
}


type Config struct {
	Enabled bool `yaml:"enabled"`
}

func (c *Config) Parse() (*KeepAlive, error) {
	return NewKeepAlive()
}

func NewKeepAlive() (*KeepAlive, error) {
	return &KeepAlive{
		Enabled: true,
	}, nil
}

func NewDefaultConfig() *Config {
	return &Config{
		Enabled: DefaultKeepAliveEnabled,
	}
}

func AddKeepAliveFlag() *Config {
	// not configurable
	return NewDefaultConfig()
}

func SetDefaultKeepAlive(conn net.Conn) error {
	return keepAlive(conn, DefaultKeepAliveEnabled)
}

func keepAlive(conn net.Conn, enabled bool) error {
	c, ok := conn.(*net.TCPConn)
	if !ok {
		return fmt.Errorf("Bad connection type: %T", c)
	}

	if err := c.SetKeepAlive(true); err != nil {
		return err
	}

	return nil
}
