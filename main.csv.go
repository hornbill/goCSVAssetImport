package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

func getAssetsFromCSV(csvFile, assetType string) (bool, []map[string]string) {
	rows := []map[string]string{}
	file, err := os.Open(csvFile)
	if err != nil {
		// err is printable
		// elements passed are separated by space automatically
		logger(4, "Error opening CSV file: "+fmt.Sprintf("%v", err), true)
		return false, rows
	}
	// automatically call Close() at the end of current method
	defer file.Close()
	//
	r := csv.NewReader(file)

	var header []string
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger(4, "Error reading CSV data: "+fmt.Sprintf("%v", err), true)
			return false, rows
		}
		if header == nil {
			header = record
		} else {
			dict := map[string]string{}
			for i := range header {
				dict[header[i]] = record[i]
			}
			rows = append(rows, dict)
		}
	}
	return true, rows

}
