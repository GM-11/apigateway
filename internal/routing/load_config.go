package routing

import (
	"log"
	"os"

	"example.com/m/v2/internal/circuitbreaker"
	"example.com/m/v2/internal/utils"
	"gopkg.in/yaml.v3"
)

func LoadConfig(path string) (*utils.Config, error) {

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Error reading file: %s", err.Error())
		return nil, err
	}
	var config utils.Config

	err = yaml.Unmarshal(data, &config)

	if err != nil {
		log.Printf("Error unmarshaling: %s", err.Error())
		return nil, err
	}

	for i := range config.Routes {
		for j := range config.Routes[i].Upstreams {
			config.Routes[i].Upstreams[j].CircuitBreaker = &circuitbreaker.CircuitBreaker{}
		}
	}

	return &config, nil
}
