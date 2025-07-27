package main

import (
	"fmt"
	"log"

	"github.com/AtoyanMikhail/auth/internal/config"
)

func main() {
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err.Error())
	}
	fmt.Println(cfg)
}
