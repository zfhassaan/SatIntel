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

// setEnvironmentalVariable prompts the user to enter a value for the given environment variable.
// It reads from stdin and sets the environment variable with the provided value.
func setEnvironmentalVariable(envKey string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", envKey)
	input, err := reader.ReadString('\n')

	if err != nil {
		fmt.Println("Error reading input:", err)
		os.Exit(1)
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
