package cli

import (
	"bufio"
	"io"
	"strings"
)

// Read parses cli args into an array of strings
func Read(r io.Reader) []string {
	args := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		args = append(args, strings.SplitN(line, " ", 2)...)
	}
	return args
}
