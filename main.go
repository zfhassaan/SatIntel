/*
Copyright © 2023 Angelina Tsuboi angelinatsuboi@proton.me
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/ANG13T/SatIntel/cli"
	"golang.org/x/term"
)

// loadEnvFile reads environment variables from a .env file in the current directory.
// It skips empty lines and comments, and handles quoted values.
func loadEnvFile() error {
	envPath := ".env"

	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		return fmt.Errorf(".env file not found")
	}

	file, err := os.Open(envPath)
	if err != nil {
		return fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			value = strings.Trim(value, "\"'")
			os.Setenv(key, value)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading .env file: %w", err)
	}

	return nil
}

// isPasswordField determines if the environment variable should have masked input.
func isPasswordField(envKey string) bool {
	passwordFields := []string{
		"SPACE_TRACK_PASSWORD",
		"N2YO_API_KEY",
		"PASSWORD",
		"API_KEY",
		"SECRET",
		"TOKEN",
	}
	envKeyUpper := strings.ToUpper(envKey)
	for _, field := range passwordFields {
		if strings.Contains(envKeyUpper, field) {
			return true
		}
	}
	return false
}

// readPassword reads a password from stdin and displays asterisks for each character typed.
func readPassword() (string, error) {
	fd := int(os.Stdin.Fd())

	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		// Fallback to regular input if not a terminal
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(input), nil
	}

	// Save current terminal state and set to raw mode
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("failed to set raw terminal: %w", err)
	}
	defer term.Restore(fd, oldState)

	var password []byte
	var input [1]byte

	for {
		n, err := os.Stdin.Read(input[:])
		if err != nil || n == 0 {
			break
		}

		char := input[0]

		// Handle Enter key (carriage return or newline)
		if char == '\r' || char == '\n' {
			fmt.Println()
			break
		}

		// Handle Backspace/Delete (127 = DEL, 8 = BS)
		if char == 127 || char == 8 {
			if len(password) > 0 {
				password = password[:len(password)-1]
				// Move cursor back, print space, move cursor back again
				fmt.Print("\b \b")
			}
			continue
		}

		// Handle Ctrl+C
		if char == 3 {
			fmt.Println()
			os.Exit(1)
		}

		// Skip control characters except those we handle
		if char < 32 {
			continue
		}

		// Add character to password and print asterisk
		password = append(password, char)
		fmt.Print("*")
	}

	return string(password), nil
}

// setEnvironmentalVariable prompts the user to enter a value for the given environment variable.
// It reads from stdin and sets the environment variable with the provided value.
// Password fields display asterisks (*) for each character typed for security.
func setEnvironmentalVariable(envKey string) string {
	var input string
	var err error

	fmt.Printf("%s: ", envKey)

	if isPasswordField(envKey) {
		input, err = readPassword()
		if err != nil {
			fmt.Println("Error reading password:", err)
			os.Exit(1)
		}
	} else {
		reader := bufio.NewReader(os.Stdin)
		input, err = reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			os.Exit(1)
		}
		input = strings.TrimSpace(input)
	}

	input = strings.TrimSpace(input)

	if err := os.Setenv(envKey, input); err != nil {
		fmt.Printf("Error setting environment variable %s: %v\n", envKey, err)
		os.Exit(1)
	}

	return input
}

// checkEnvironmentalVariable verifies if an environment variable exists.
// If it doesn't exist, it prompts the user to provide a value.
func checkEnvironmentalVariable(envKey string) {
	_, found := os.LookupEnv(envKey)
	if !found {
		setEnvironmentalVariable(envKey)
	}
}

func main() {
	err := loadEnvFile()
	if err != nil {
		if err.Error() == ".env file not found" {
			fmt.Println("Note: .env file not found. Please provide credentials:")
		} else {
			fmt.Printf("Warning: Error loading .env file: %v\n", err)
			fmt.Println("Please provide credentials manually:")
		}
		fmt.Println()
	} else {
		fmt.Println("Loaded credentials from .env file")
	}

	checkEnvironmentalVariable("SPACE_TRACK_USERNAME")
	checkEnvironmentalVariable("SPACE_TRACK_PASSWORD")
	checkEnvironmentalVariable("N2YO_API_KEY")

	cli.SatIntel()
}
