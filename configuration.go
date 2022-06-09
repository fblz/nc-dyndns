package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type Configuration struct {
	hosts []Host
}

// ReadConfig reads the configuration file
func NewConfiguration(configPath string) (*Configuration, error) {
	confBytes, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var hosts []Host
	err = json.Unmarshal(confBytes, &hosts)
	if err != nil {
		return nil, err
	}

	this := new(Configuration)

	for i := range hosts {

		if strings.TrimSpace(hosts[i].Hostname) == "" {
			fmt.Printf("Record %d (%s) has errors and is not loaded:\n", i, hosts[i].FQDN())
			fmt.Println("Hostname property not set")
		}

		if strings.TrimSpace(hosts[i].Zone) == "" {
			fmt.Printf("Record %d (%s) has errors and is not loaded:\n", i, hosts[i].FQDN())
			fmt.Println("Zone property not set")
		}

		if strings.TrimSpace(hosts[i].APIToken) == "" {
			fmt.Printf("Record %d (%s) has errors and is not loaded:\n", i, hosts[i].FQDN())
			fmt.Println("APIToken property not set")
		}

		err := hosts[i].Populate()
		if err != nil {
			fmt.Printf("Record %d (%s) has errors and is not loaded:\n", i, hosts[i].FQDN())
			fmt.Println(err)
			continue
		}
		this.hosts = append(this.hosts, hosts[i])
	}

	confBytes, err = json.MarshalIndent(hosts, "", "\t")
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(configPath, confBytes, 0600)
	if err != nil {
		return nil, err
	}

	if len(this.hosts) == 0 {
		return nil, fmt.Errorf("no records were loaded")
	}

	return this, nil
}

func (c *Configuration) GetHost(fqdn string) (*Host, error) {
	for i := range c.hosts {
		if c.hosts[i].FQDN() == fqdn {
			return &c.hosts[i], nil
		}
	}

	return nil, fmt.Errorf("unknown fqdn %s", fqdn)
}
