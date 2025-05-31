package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <input-coverage-file> <output-coverage-file>\n", os.Args[0])
		os.Exit(1)
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	input, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
		os.Exit(1)
	}
	defer input.Close()

	output, err := os.Create(outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer output.Close()

	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for scanner.Scan() {
		line := scanner.Text()

		if shouldExclude(line) {
			continue
		}

		fmt.Fprintln(writer, line)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "Error reading input file: %v\n", err)
		os.Exit(1)
	}
}

func shouldExclude(line string) bool {
	// Keep the mode line
	if strings.HasPrefix(line, "mode:") {
		return false
	}

	excludePatterns := []string{
		"_test.go:",
		"/mocks/",
		"/migrations/",
		"main.go:",
		"/cmd/",
		"/integration_tests/",
		"/testutils/",
		"benchmark_test.go:",
		".test",
		"coverage.out",
		"cpu.out",
		"mem.out",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}
