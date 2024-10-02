package main

import (
	"log"
	"os"

	"github.com/cofide/cofidectl/cmd/cofidectl/cmd"
)

func main() {
	rootCmd, err := cmd.NewRootCmd(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}

	if err = rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
