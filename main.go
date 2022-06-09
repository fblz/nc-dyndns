package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
)

func main() {
	config := flag.String("config", "", "Path to the config.json file.")
	help := flag.Bool("help", false, "Print this message.")

	flag.Parse()

	if *help || *config == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	handler, err := NewHandler(*config)
	if err != nil {
		panic(err)
	}

	fmt.Println("Loaded")

	http.HandleFunc("/dyndns", handler.DynDNSHandler)
	err = http.ListenAndServe("127.0.1.1:8080", nil)

	if err != nil {
		panic(err)
	}

	fmt.Println("Bye")
}
