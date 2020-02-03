package main

import (
	"crypto/rand"
	"encoding/base64"
	"flag"
	"fmt"
	"nc-dyndns/netcupapi"
	"net/http"
	"os"
)

var config string
var help bool
var newtoken bool

func init() {
	flag.StringVar(&config, "config", "", "Path to the config.json file.")
	flag.BoolVar(&help, "help", false, "Print this message.")
	flag.BoolVar(&newtoken, "newtoken", false, "Output a new secure API token and exit.")
}

func generateAPIKey(byteLength int) (*string, error) {
	random := make([]byte, byteLength)
	n, err := rand.Read(random)
	if err != nil {
		return nil, err
	}

	if n != byteLength {
		return nil, fmt.Errorf("Bad randomness")
	}

	encoded := base64.RawURLEncoding.EncodeToString(random)
	return &encoded, nil
}

func main() {
	flag.Parse()

	if help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	if newtoken {
		token, err := generateAPIKey(32)

		if err != nil {
			fmt.Println(err)
			os.Exit(3)
		}

		fmt.Println(*token)
		os.Exit(0)
	}

	if config == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	nc, err := netcupapi.New(config)

	if err != nil {
		panic("Could not read configuration, aborting.")
	}

	http.HandleFunc("/dyndns", nc.DynDNSHandler)
	err = http.ListenAndServe("127.0.1.1:8080", nil)

	if err != nil {
		panic(err)
	}
}
