package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	fmt.Println("=== Running All User Examples ===")
	fmt.Println()

	// Get the current directory (should be examples/user)
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatal("Failed to get current directory:", err)
	}

	// Find all main.go files in subdirectories
	examples, err := findExamples(currentDir)
	if err != nil {
		log.Fatal("Failed to find examples:", err)
	}

	if len(examples) == 0 {
		fmt.Println("No user examples found in subdirectories")
		return
	}

	fmt.Printf("Found %d user examples to run:\n", len(examples))
	for _, example := range examples {
		fmt.Printf("  - %s\n", example)
	}
	fmt.Println()

	// Track results
	results := make(map[string]bool)
	var totalDuration time.Duration

	// Run each example
	for i, example := range examples {
		fmt.Printf("[%d/%d] Running %s...\n", i+1, len(examples), example)

		start := time.Now()
		success := runExample(example)
		duration := time.Since(start)
		totalDuration += duration

		results[example] = success

		status := "✅ SUCCESS"
		if !success {
			status = "❌ FAILED"
		}

		fmt.Printf("[%d/%d] %s - %s (took %v)\n", i+1, len(examples), example, status, duration)
		fmt.Println(strings.Repeat("-", 60))
	}

	// Print summary
	fmt.Println()
	fmt.Println("=== SUMMARY ===")
	fmt.Printf("Total execution time: %v\n", totalDuration)
	fmt.Println()

	successCount := 0
	failedCount := 0

	for example, success := range results {
		status := "✅"
		if !success {
			status = "❌"
			failedCount++
		} else {
			successCount++
		}
		fmt.Printf("%s %s\n", status, example)
	}

	fmt.Println()
	fmt.Printf("Results: %d successful, %d failed, %d total\n", successCount, failedCount, len(examples))

	if failedCount > 0 {
		fmt.Println("\n⚠️  Some examples failed. Check the logs above for details.")
		os.Exit(1)
	} else {
		fmt.Println("\n🎉 All user examples completed successfully!")
	}
}

// findExamples searches for all subdirectories containing main.go files
func findExamples(baseDir string) ([]string, error) {
	var examples []string

	// Read all entries in the current directory
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Skip hidden directories and the current directory name itself
			if strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			// Check if this directory contains a main.go file
			mainGoPath := filepath.Join(baseDir, entry.Name(), "main.go")
			if _, err := os.Stat(mainGoPath); err == nil {
				examples = append(examples, entry.Name())
			}
		}
	}

	return examples, nil
}

// runExample executes a single example and returns whether it succeeded
func runExample(exampleName string) bool {
	fmt.Printf("  → Starting %s\n", exampleName)

	// Change to the example directory
	exampleDir := filepath.Join(".", exampleName)

	// Run go run . in the example directory to include all .go files
	cmd := exec.Command("go", "run", ".")
	cmd.Dir = exampleDir

	// Capture both stdout and stderr
	output, err := cmd.CombinedOutput()

	if err != nil {
		fmt.Printf("  ❌ %s failed with error: %v\n", exampleName, err)
		fmt.Printf("  📄 Output:\n%s\n", string(output))
		return false
	}

	fmt.Printf("  ✅ %s completed successfully\n", exampleName)

	// Show first few lines of output for verification
	outputLines := strings.Split(string(output), "\n")
	fmt.Printf("  📄 Output preview:\n")
	for i, line := range outputLines {
		if i >= 3 { // Show only first 3 lines
			if len(outputLines) > 3 {
				fmt.Printf("    ... (%d more lines)\n", len(outputLines)-3)
			}
			break
		}
		if line != "" {
			fmt.Printf("    %s\n", line)
		}
	}

	return true
}
