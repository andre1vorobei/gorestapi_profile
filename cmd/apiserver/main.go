package main

import (
	"flag"
	"fmt"
	"gorestapi/internal/app/apiserver"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

var (
	configPath string
)

func init() {
	flag.StringVar(&configPath, "config-path", "configs/apiserver.toml", "path to config")
}

func main() {

	flag.Parse()

	config := apiserver.NewConfig()

	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env file")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatal(err)
	}

	// Подставляем переменные окружения
	expandedConfig := os.ExpandEnv(string(data))
	fmt.Println(expandedConfig)
	//_, err := toml.DecodeFile(configPath, config)
	_, err = toml.Decode(expandedConfig, config)

	if err != nil {
		log.Fatal(err)
	}

	s := apiserver.New(config)

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}

}
