package jsonData

import (
	"encoding/json"
	"io"
	"os"

	"crawler/src/common"

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

func LoadRobotsMap() (map[string]string, error) {
	// Open the JSON file
	file, err := os.Open("jsonData/robots_txt.json")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Read the contents of the file
	byteValue, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var robotsList []common.RobotsItem
	err = json.Unmarshal(byteValue, &robotsList)
	if err != nil {
		return nil, err
	}

	robotsMap := common.RobotsListToMap(robotsList)

	return robotsMap, nil
}

func DumpRobots(robotsMap map[string]string) error {
	robotsList := common.RobotsMapToList(robotsMap)

	// Serialize the list to JSON
	data, err := json.MarshalIndent(robotsList, "", "  ")
	if err != nil {
		return err
	}

	// Write JSON data to the file
	err = os.WriteFile("jsonData/robots_txt.json", data, 0644)
	if err != nil {
		return err
	}

	return nil
}
