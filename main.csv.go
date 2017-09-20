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

    bom := make([]byte, 3)
    file.Read(bom)
    if bom[0] == 0xEF && bom[1] == 0xBB && bom[2] == 0xBF {
        // BOM Detected, continue with feeding the file fmt.Println("BOM")
    } else {
        // No BOM Detected, reset the file feed
        file.Seek(0,0)
    }

    //because the json configuration loader cannot handle runes, code here to convert string to rune-array and getting first item
	r := csv.NewReader(file)
	if CSVImportConf.CSVCommaCharacter != "" {
        CSVCommaRunes := []rune(CSVImportConf.CSVCommaCharacter)
        r.Comma = CSVCommaRunes[0]
        //r.Comma = ';'
    }
    
	if CSVImportConf.CSVLazyQuotes {
        r.LazyQuotes = true
    }
	if CSVImportConf.CSVFieldsPerRecord > 0 {
        r.FieldsPerRecord = CSVImportConf.CSVFieldsPerRecord
    }
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
