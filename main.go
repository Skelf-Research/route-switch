package main

import (
	"fmt"
	"log"
	"os"

	"github.com/skelf-research/route-switch/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
	fmt.Println("Thank you for using route-switch!")
}