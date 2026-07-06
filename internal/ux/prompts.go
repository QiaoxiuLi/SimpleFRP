package ux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Ask(prompt string) (string, error) {
	fmt.Println(prompt)
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	return strings.TrimSpace(text), err
}
