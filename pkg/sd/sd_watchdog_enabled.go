package sd

import (
	"errors"
	"fmt"
	"os"
	"time"
)

// WatchdogEnabled checks whether the service manager expects watchdog keep-alive notifications and
// returns the timeout value in µs. A timeout value of 0 signifies no notifications are expected.
// http://www.freedesktop.org/software/systemd/man/sd_watchdog_enabled.html
func WatchdogEnabled() (time.Duration, error) {
	spid := os.Getenv("WATCHDOG_PID")
	if spid != "" {
		pid := 0
		n, err := fmt.Sscanf(spid, "%d", &pid)
		if err != nil {
			return 0, err
		}
		if n != 1 {
			return 0, errors.New("could not scan WATCHDOG_PID")
		}
		if pid != os.Getpid() {
			return 0, nil
		}
	}

	sttl := os.Getenv("WATCHDOG_USEC")
	if sttl == "" {
		return 0, errors.New("missing WATCHDOG_USEC")
	}
	ttl := uint64(0)
	n, err := fmt.Sscanf(sttl, "%d", &ttl)
	if err != nil {
		return 0, err
	}
	if n != 1 {
		return 0, errors.New("could not scan WATCHDOG_USEC")
	}
	return time.Duration(ttl) * time.Microsecond, nil
}
