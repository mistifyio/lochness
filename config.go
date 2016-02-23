package lochness

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

//Used to get set arbitrary config variables

var (
	// ConfigPath is the path in the config store.
	ConfigPath = "/lochness/config/"
)

// GetConfig gets a single value from the config store. The key can contain slashes ("/")
func (c *Context) GetConfig(key string) (string, error) {
	if key == "" {
		return "", errors.New("empty config key")
	}

	resp, err := c.etcd.Get(filepath.Join(ConfigPath, key), false, false)
	if err != nil {
		return "", err
	}

	return resp.Node.Value, nil

}

// SetConfig sets a single value from the config store. The key can contain slashes ("/")
func (c *Context) SetConfig(key, val string) error {
	if key == "" {
		return errors.New("empty config key")
	}

	_, err := c.etcd.Set(filepath.Join(ConfigPath, key), val, 0)
	return err
}

// ForEachConfig will run f on each config. It will stop iteration if f returns an error.
func (c *Context) ForEachConfig(f func(key, val string) error) error {
	resp, err := c.etcd.Get(ConfigPath, true, true)
	if err != nil {
		return err
	}
	return forEachConfig(resp.Node.Nodes, f)
}

func forEachConfig(nodes etcd.Nodes, f func(key, val string) error) error {
	for _, n := range nodes {
		if n.Dir {
			if err := forEachConfig(n.Nodes, f); err != nil {
				return err
			}
		} else {
			k := strings.TrimPrefix(n.Key, ConfigPath)
			v := n.Value
			if err := f(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
