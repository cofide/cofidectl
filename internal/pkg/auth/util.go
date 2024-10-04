package auth

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	credentialsDirectory string = "/.cofide"
	credentialsFileKey   string = "cofide_access_token"
	credentialsFilePath  string = "/.cofide/credentials"
)

func GetTokenFromCredentialsFile() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	credentialsFullFilePath := fmt.Sprintf("%s%s", homeDir, credentialsFilePath)
	openCredentialsFile, err := os.OpenFile(credentialsFullFilePath, os.O_RDONLY, 0)
	if err != nil {
		return "", nil
	}
	defer openCredentialsFile.Close()

	scanner := bufio.NewScanner(openCredentialsFile)

	if scanner.Scan() {
		// scan the first (and only - assumed) line
		line := scanner.Text()

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			if key == credentialsFileKey {
				token := strings.TrimSpace(parts[1])
				return token, nil
			}
		}
		return "", fmt.Errorf("invalid format")
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	return "", fmt.Errorf("file is empty")

}
