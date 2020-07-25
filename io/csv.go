package io

import (
	"encoding/csv"
	"errors"
	"log"
	"net/http"
	"os"
	"strconv"
)

// ReadCsvByName accepts the file path and returns the pointer of Checkpoint list
func ReadCsvByName(fileLoc string) (cpListPtr *[]Checkpoint, errCode int, err error) {
	csvFile, err := os.Open(fileLoc)
	defer csvFile.Close()
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusNotFound, err
	}

	return ReadCsvByPtr(csvFile)
}

// ReadCsvByPtr accepts the file pointer and returns the pointer of Checkpoint list
func ReadCsvByPtr(csvFile *os.File) (cpListPtr *[]Checkpoint, errCode int, err error) {

	csvFile.Seek(0, 0)
	csvReader := csv.NewReader(csvFile)
	defer csvFile.Close()

	if _, err := csvReader.Read(); err != nil { // Skip the file header
		log.Println(err)
	}
	rows, err := csvReader.ReadAll() // rows: [][]string
	if err != nil {
		log.Println(err.Error())
		return nil, http.StatusNotAcceptable, err
	}

	// Logic: Read all the valid lines and omit invalid ones.
	// Line structure: name, base, rx, ry, isPortal
	cpList := make([]Checkpoint, 0, len(rows))
	omitCounter := 0
	parseF64 := func(raw string) (ans float64, err error) {
		ans, err = strconv.ParseFloat(raw, 64)
		if err != nil {
			log.Println(err)
			omitCounter++
			return 0, err
		}
		return ans, nil
	}

	for _, row := range rows {
		var tempCP Checkpoint
		var tempF64 float64
		var tempBool bool

		tempCP.Name = row[0]
		tempCP.Base = row[1]
		tempF64, err = parseF64(row[2]) // rx
		if err != nil {
			continue
		} else {
			tempCP.Rx = tempF64
		}

		tempF64, err = parseF64(row[3]) // ry
		if err != nil {
			continue
		} else {
			tempCP.Ry = tempF64
		}

		tempBool, err = strconv.ParseBool(row[4])
		if err != nil {
			log.Println(err.Error())
			omitCounter++
			continue
		} else {
			tempCP.IsPortal = tempBool
		}

		if tempCP.IsPortal == false { // ignore the weight of a space, default 0
			if tWeight := row[5]; tWeight == "" {
				// is an Asset, assign default weight
				tempCP.Weight = 1
			} else {
				tempF64, err = parseF64(row[5]) // weight
				if err != nil {
					continue
				} else if tempF64 < 0 {
					err = errors.New("sampling weight cannot be negative")
					omitCounter++
					continue
				} else {
					tempCP.Weight = tempF64
				}
			}
		}

		cpList = append(cpList, tempCP)
	}

	if omitCounter == 0 {
		return &cpList, http.StatusCreated, nil
	}
	return &cpList, http.StatusPartialContent, errors.New(string(omitCounter) + " lines cannot be parsed")
}
