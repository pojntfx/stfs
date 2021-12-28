package main

import "github.com/pojntfx/stfs/cmd/stfs/cmd"

func main() {
	if err := cmd.Execute(); err != nil {
		panic(err)
	}
}
