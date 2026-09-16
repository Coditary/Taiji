package main

import (
	"github.com/coditary/taiji/internal/cli"
)

func main() {
	app, err := cli.New()
	if err != nil {
		panic(err)
	}
	app.Execute()
}
