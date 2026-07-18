package main

import (
	"github.com/bizshuk/port/cmd"
	"github.com/bizshuk/port/config"
)

func main() {
	config.Default()
	cmd.Execute()
}
