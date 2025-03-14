package autocert

import (
	"encoding/csv"
	"fmt"
	"os"
)

// ReadCSV reads and parses a CSV file, returning the data as a slice of string slices.
// Each inner slice represents a row of the CSV.
func ReadCSV(filename string) ([][]string, error) {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)

	// Read all records
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("error reading CSV: %w", err)
	}

	return records, nil
}

// Reads a CSV file and returns the data as a slice of maps.
// The first row is assumed to be the header, and its values are used as keys.
//
//	call records, err := ReadCSV(filename)
//
//	if err != nil {
//		return nil, err
//	}
func ParseCSVToMap(records [][]string) ([]map[string]string, error) {
	if len(records) == 0 {
		return []map[string]string{}, nil
	}

	headers := records[0]
	headerCount := make(map[string]int)

	// Check for duplicate headers and rename them
	for i, header := range headers {
		if count, exists := headerCount[header]; exists {
			headerCount[header]++
			headers[i] = fmt.Sprintf("%s_%d", header, count+1)
		} else {
			headerCount[header] = 0
		}
	}

	result := make([]map[string]string, 0, len(records)-1)

	for i := 1; i < len(records); i++ {
		row := make(map[string]string)
		// Fill the map with header keys and corresponding values
		for j := 0; j < len(headers); j++ {
			if j < len(records[i]) {
				row[headers[j]] = records[i][j]
			} else {
				// Handle missing values
				row[headers[j]] = ""
			}
		}
		result = append(result, row)
	}

	return result, nil
}
