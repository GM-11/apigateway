package routing

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file: %s", err.Error())
		return nil, err
	}
	var config Config

	err = yaml.Unmarshal(data, &config)

	if err != nil {
		log.Printf("Error unmarshaling: %s", err.Error())
		return nil, err
	}

	return &config, nil
}
