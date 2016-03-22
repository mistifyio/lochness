package lochness

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/mistifyio/lochness/pkg/kv"
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

	resp, err := c.kv.Get(filepath.Join(ConfigPath, key))
	if err != nil {
		return "", err
	}

	return string(resp.Data), nil

}

// SetConfig sets a single value from the config store. The key can contain slashes ("/")
func (c *Context) SetConfig(key, val string) error {
	if key == "" {
		return errors.New("empty config key")
	}

	err := c.kv.Set(filepath.Join(ConfigPath, key), val)
	return err
}

// ForEachConfig will run f on each config. It will stop iteration if f returns an error.
func (c *Context) ForEachConfig(f func(key, val string) error) error {
	nodes, err := c.kv.GetAll(ConfigPath)
	if err != nil {
		return err
	}
	return forEachConfig(nodes, f)
}

func forEachConfig(nodes map[string]kv.Value, f func(key, val string) error) error {
	for k, v := range nodes {
		k = strings.TrimPrefix(k, ConfigPath)
		if err := f(k, string(v.Data)); err != nil {
			return err
		}
	}
	return nil
}
