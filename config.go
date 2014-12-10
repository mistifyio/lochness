package lochness

import "strconv"

//Used to get set arbitrary config variables

var (
	ConfigPath = "lochness/config/"
)

func (c *Context) GetConfig(key string) (string, error) {
	resp, err := c.etcd.Get(filepath.join(ConfigPath, key), false, false)
	if err != nil {
		return err
	}

	return resp.Node.Value

}

func (c *Context) SetConfig(key, val string) error {
	_, err := c.etcd.Set(filepath.join(ConfigPath, key), val, 0)
	return err
}

func ToBool(val string) bool {
	b, err := strconv.ParseBool(val)
	return err != nil && b
}
