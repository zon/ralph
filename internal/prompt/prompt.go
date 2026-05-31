package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func ProviderKey(provider string) (string, error) {
	fmt.Fprintf(os.Stdout, "Enter API key for %s: ", provider)

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}

	key := strings.TrimSpace(line)
	if key == "" {
		return "", fmt.Errorf("API key for %s cannot be blank", provider)
	}

	return key, nil
}
