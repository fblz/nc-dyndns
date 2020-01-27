package main

import (
	"flag"
	"nc-dyndns/netcupapi"
	"net/http"
	"os"
)

func main() {
	var config string
	flag.StringVar(&config, "config", "", "Path to the config.json file.")
	flag.Parse()

	if config == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	nc, err := netcupapi.New(config)

	if err != nil {
		panic("Could not read configuration, aborting.")
	}

	http.HandleFunc("/dyndns", nc.DynDNSHandler)
	err = http.ListenAndServe(":8080", nil)

	if err != nil {
		panic(err)
	}
}
