package main

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
)

type Handler struct {
	configuration Configuration
}

func NewHandler(configPath string) (*Handler, error) {
	configuration, err := NewConfiguration(configPath)
	if err != nil {
		return nil, err
	}

	this := new(Handler)
	this.configuration = *configuration
	return this, nil
}

func validateQuery(query url.Values) (map[string]string, error) {
	params := []string{"fqdn", "ipv4", "token"}
	validQuery := make(map[string]string)

	for _, param := range params {
		queryItems, ok := query[param]

		if !ok || len(queryItems) < 1 {
			return nil, fmt.Errorf("parameter %s is missing", param)
		}

		validQuery[param] = queryItems[0]
	}

	return validQuery, nil
}

// DynDNSHandler provides a handler function for the web package
func (h *Handler) DynDNSHandler(w http.ResponseWriter, r *http.Request) {
	query, err := validateQuery(r.URL.Query())

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Missing or invalid parameter(s)", http.StatusBadRequest)
		return
	}

	//validate ip by parsing it in go
	ip := net.ParseIP(query["ipv4"])

	if ip == nil {
		fmt.Printf("Invalid IP Address %s\n", query["ipv4"])
		http.Error(w, "Missing or invalid parameter(s)", http.StatusBadRequest)
		return
	}

	//validate ipv4 by converting
	ip = ip.To4()

	if ip == nil {
		fmt.Printf("IP Address %s is not v4\n", query["ipv4"])
		http.Error(w, "Missing or invalid parameter(s)", http.StatusBadRequest)
		return
	}

	//validate fqdn by searching it in the config file
	host, err := h.configuration.GetHost(query["fqdn"])

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Not authorized to set this fqdn", http.StatusForbidden)
		return
	}

	//TODO: Maybe fix time-based enum of FQDNs
	//validate token by searching in the config file
	if host.WebToken != query["token"] {
		fmt.Printf("Invalid token supplied for fqdn %s\n", host.FQDN())
		http.Error(w, "Not authorized to set this fqdn", http.StatusForbidden)
		return
	}

	if host.IP == ip.String() {
		fmt.Println("IP did not change")
		fmt.Fprint(w, "success")
		return
	}

	fmt.Printf("Changing IP Address from %v to %v\n", host.IP, ip.String())

	err = host.updateIP(ip)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something unexpected happened", http.StatusInternalServerError)
		return
	}

	fmt.Println("Success")
	fmt.Fprint(w, "success")
}
