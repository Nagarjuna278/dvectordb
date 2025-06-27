package kvstore

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const dataFilePath = "kvstore.data"

func saveMapToFile(data map[string]string) error {
	file, err := os.Create(dataFilePath) // os.Create truncates the file if it exists, effectively "replacing"
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for key, value := range data {
		_, err := writer.WriteString(fmt.Sprintf("%s:%s\n", key, value))
		if err != nil {
			return fmt.Errorf("failed to write to file: %w", err)
		}
	}
	return nil
}

func LoadFileToMap() (*map[string]string, error) {
	file, err := os.OpenFile(dataFilePath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	data := make(map[string]string)

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line format: %s", line)
		}
		data[parts[0]] = parts[1]
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &data, nil
}
