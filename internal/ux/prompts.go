package ux

import (
	"fmt"
	"golang.org/x/term"
	"io"
	"os"
	"strings"
)

func Ask(prompt string) (string, error) {
	fmt.Println(prompt)
	var text strings.Builder
	var b [1]byte
	for text.Len() < 4096 {
		n, err := os.Stdin.Read(b[:])
		if n > 0 {
			if b[0] == '\n' {
				return strings.TrimSpace(text.String()), nil
			}
			text.WriteByte(b[0])
		}
		if err != nil {
			if err == io.EOF && text.Len() > 0 {
				return strings.TrimSpace(text.String()), nil
			}
			return "", err
		}
	}
	return "", fmt.Errorf("input exceeds 4096 bytes")
}

// Interactive secrets are never echoed; piped input remains usable for automation.
func AskSecret(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return Ask(prompt)
	}
	fmt.Println(prompt)
	data, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	defer func() {
		for i := range data {
			data[i] = 0
		}
	}()
	return strings.TrimSpace(string(data)), err
}
