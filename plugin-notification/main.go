package main

import "github.com/crawlab-team/plugin-notification/core"

func main() {
	svc := core.NewService()
	if err := svc.Start(); err != nil {
		panic(err)
	}
}
