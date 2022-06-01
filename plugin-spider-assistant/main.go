package main

import "github.com/crawlab-team/plugin-scrapy/services"

func main() {
	svc := services.NewService()
	if err := svc.Start(); err != nil {
		panic(err)
	}
}
