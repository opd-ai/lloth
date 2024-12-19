package main

import (
	"io/ioutil"
	"log"
	"regexp"
	"strings"
)

var extra = []string{"google.com", "yahoo.com", "github.com"}

func main() {
	// Read the hosts.txt file
	inputFile := "hosts.txt"
	data, err := ioutil.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Could not read file: %v", err)
	}

	// Convert data to a string
	content := string(data)

	// Remove lines starting with #
	lines := strings.Split(content, "\n")
	var cleanedLines []string

	for _, line := range lines {
		// Trim the line and check if it starts with # or is empty
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "#") || trimmedLine == "" {
			continue
		}
		cleanedLines = append(cleanedLines, trimmedLine)
	}

	// Join cleaned lines back into a single string
	content = strings.Join(cleanedLines, "\n")

	// Define regex pattern to match IPv4, IPv6, and localhost addresses and domain names
	ipPattern := regexp.MustCompile(`\b(?:\d{1,3}(?:\.\d{1,3}){3}|(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,7}:|(?:[0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|(?:[0-9a-fA-F]{1,4}:){1,5}(?::[0-9a-fA-F]{1,4}){1,2}|(?:[0-9a-fA-F]{1,4}:){1,4}(?::[0-9a-fA-F]{1,4}){1,3}|(?:[0-9a-fA-F]{1,4}:){1,3}(?::[0-9a-fA-F]{1,4}){1,4}|(?:[0-9a-fA-F]{1,4}:){1,2}(?::[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:(?::[0-9a-fA-F]{1,4}){1,6}|:((?::[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(?::[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::1|127\.0\.0\.1|localhost)\b`)

	// Remove IP addresses and localhost domain names
	content = ipPattern.ReplaceAllString(strings.Join(cleanedLines, "\n"), "")

	// Remove spaces and tabs
	content = strings.ReplaceAll(content, " ", "")
	content = strings.ReplaceAll(content, "\t", "")

	// Remove any remaining empty lines
	lines = strings.Split(content, "\n")
	var finalLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "::") {
			finalLines = append(finalLines, line)
		}
	}
	for _, line := range extra {
		if strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "::") {
			finalLines = append(finalLines, line)
		}
	}

	// Join final cleaned lines back into a single string
	content = strings.Join(finalLines, "\n")

	// Optionally, write the cleaned content back to a new file
	outputFile := "cleaned_hosts.txt"
	err = ioutil.WriteFile(outputFile, []byte(content), 0644)
	if err != nil {
		log.Fatalf("Could not write to file: %v", err)
	}

	log.Printf("Cleaned content written to %s", outputFile)
}
