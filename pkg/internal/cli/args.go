package cli

import (
	"bufio"
	"io"
	"strings"
)

func Read(r io.Reader) []string {
	args := []string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		args = append(args, strings.SplitN(line, " ", 2)...)
	}
	return args
}
