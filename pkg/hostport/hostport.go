// Package hostport provides more sane and expected behavior for splitting a
// network address into host and port parts
package hostport

import (
	"errors"
	"net"
	"strings"
)

const missingPortMsg = "missing port in address"

// Split splits a network address of the form "host", "host:port", "[host]",
// "[host]:port", "[ipv6-host%zone]", or "[ipv6-host%zone]:port" into host or
// ipv6-host%zone and port. Port will be an empty string if not supplied.
func Split(hostport string) (host string, port string, err error) {
	var rawport string

	if len(hostport) == 0 {
		return
	}

	// Limit literal brackets to max one open and one closed
	openPos := strings.Index(hostport, "[")
	if openPos != strings.LastIndex(hostport, "[") {
		err = errors.New("too many '['")
		return
	}
	closePos := strings.Index(hostport, "]")
	if closePos != strings.LastIndex(hostport, "]") {
		err = errors.New("too many ']'")
		return
	}

	// Break into host and port parts based on literal brackets
	if openPos > -1 {
		// Needs to open with the '['
		if openPos != 0 {
			err = errors.New("nothing can come before '['")
			return
		}
		// Must have a matching ']'
		if closePos == -1 {
			err = errors.New("missing ']'")
			return
		}
		host = hostport[1:closePos]
		rawport = hostport[closePos+1:]
	} else if closePos > -1 {
		// Did not have a matching '['
		err = errors.New("missing '['")
		return
	} else {
		// No literal brackets, split on the last :
		splitPos := strings.LastIndex(hostport, ":")
		if splitPos < 0 {
			host = hostport
		} else {
			host = hostport[0:splitPos]
			rawport = hostport[splitPos:]
		}
	}

	if rawport != "" {
		if strings.LastIndex(rawport, ":") != 0 {
			err = errors.New("poorly separated or formatted port")
			return
		}
		port = rawport[1:]
	}
	return
}

func isMissingPort(err error) bool {
	addrError, ok := err.(*net.AddrError)
	if !ok {
		return false
	}
	return addrError.Err == missingPortMsg
}
