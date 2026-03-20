package main

import (
	"log"

	"pine-ai/router"
)

func main() {

	r := router.InitRouter()

	log.Printf("server listening on %s", ":8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatalf("server start failed: %v", err)
	}
}
