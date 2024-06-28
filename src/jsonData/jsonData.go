package jsonData

import (
	"encoding/json"
	"io"
	"os"

	"github.com/charmbracelet/log"
)

type SeedList struct {
	List []string `json:"seed_list"`
}

func LoadSeedList() ([]string, error) {
	// Open the JSON file
	file, err := os.Open("jsonData/seed_list.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the contents of the file
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	// Decode the JSON data into the Go struct
	var seedList SeedList
	err = json.Unmarshal(byteValue, &seedList)
	if err != nil {
		return nil, err
	}

	// Print the loaded data
	log.Info("Successfully loaded the Seed list")

	return seedList.List, nil
}
