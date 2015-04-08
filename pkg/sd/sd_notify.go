// Package sd implements some systemd interaction, namely the equivalent of
// sd_notify and sd_watchdog_enabled
package sd

// adapted from https://raw.githubusercontent.com/docker/docker/master/pkg/systemd/sd_notify.go

import (
	"errors"
	"net"
	"os"
)

// ErrNotifyNoSocket is an error for when a valid notify socket name isn't prvided
var ErrNotifyNoSocket = errors.New("No socket")

// Notify sends a message to the init daemon. It is common to ignore the error.
func Notify(state string) error {
	socketAddr := &net.UnixAddr{
		Name: os.Getenv("NOTIFY_SOCKET"),
		Net:  "unixgram",
	}

	if socketAddr.Name == "" {
		return ErrNotifyNoSocket
	}
	switch socketAddr.Name[0] {
	case '@', '/':
	default:
		return ErrNotifyNoSocket
	}

	conn, err := net.DialUnix(socketAddr.Net, nil, socketAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(state))
	if err != nil {
		return err
	}

	return nil
}
