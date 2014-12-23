package lochness

import (
	"path/filepath"
	"strconv"
)

//Used to get set arbitrary config variables

var (
	// ConfigPath is the path in the config store.
	ConfigPath = "lochness/config/"
)

// GetConfig gets a single value from the config store. The key can contain slashes ("/")
func (c *Context) GetConfig(key string) (string, error) {
	resp, err := c.etcd.Get(filepath.Join(ConfigPath, key), false, false)
	if err != nil {
		return "", err
	}

	return resp.Node.Value, nil

}

// SetConfig sets a single value from the config store. The key can contain slashes ("/")
func (c *Context) SetConfig(key, val string) error {
	_, err := c.etcd.Set(filepath.Join(ConfigPath, key), val, 0)
	return err
}

// ToBool is a wrapper around strconv.ParseBool for easy boolean values
func ToBool(val string) bool {
	b, err := strconv.ParseBool(val)
	return err != nil && b
}
