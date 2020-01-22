package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"
)

var netClient = &http.Client{
	Timeout: time.Second * 10,
}

type answer struct {
	Serverrequestid string          `json:"serverrequestid"`
	Clientrequestid string          `json:"clientrequestid"`
	Action          string          `json:"action"`
	Status          string          `json:"status"`
	Statuscode      int             `json:"statuscode"`
	Shortmessage    string          `json:"shortmessage"`
	Longmessage     string          `json:"longmessage"`
	Responsedata    json.RawMessage `json:"responsedata"`
}

type loginAnswer struct {
	Apisessionid string `json:"apisessionid"`
}

func (a *answer) GetLoginResponse() (*loginAnswer, error) {
	var l loginAnswer
	err := json.Unmarshal(a.Responsedata, &l)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

type dnsRecord struct {
	Deleterecord bool   `json:"deleterecord"`
	Destination  string `json:"destination"`
	Hostname     string `json:"hostname"`
	ID           string `json:"id"`
	Priority     string `json:"priority"`
	State        string `json:"state"`
	Type         string `json:"type"`
}

type dnsRecordSet struct {
	Dnsrecords []dnsRecord `json:"dnsrecords"`
}

func (s *dnsRecordSet) firstOrNewRecord(hostname string, recordType string) *dnsRecord {
	for _, entry := range s.Dnsrecords {
		if entry.Hostname == hostname && entry.Type == recordType {
			return &entry
		}
	}
	return &dnsRecord{Hostname: hostname, Deleterecord: false, Type: recordType}
}

func (a *answer) GetDNSRecordResponse() (*dnsRecordSet, error) {
	var l dnsRecordSet
	err := json.Unmarshal(a.Responsedata, &l)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (a *answer) GetGenericResponse() (*map[string]interface{}, error) {
	var l map[string]interface{}
	err := json.Unmarshal(a.Responsedata, &l)

	if err != nil {
		return nil, err
	}

	return &l, nil
}

func (a *answer) GetStringResponse() string {
	return string(a.Responsedata)
}

type request struct {
	Action string      `json:"action"`
	Param  interface{} `json:"param"`
}

type login struct {
	Customernumber int    `json:"customernumber"`
	Apikey         string `json:"apikey"`
	Apipassword    string `json:"apipassword"`
}

type logout struct {
	Customernumber int    `json:"customernumber"`
	Apikey         string `json:"apikey"`
	Apisessionid   string `json:"apisessionid"`
}

type infoDNSRecords struct {
	Domainname     string `json:"domainname"`
	Customernumber int    `json:"customernumber"`
	Apikey         string `json:"apikey"`
	Apisessionid   string `json:"apisessionid"`
}

type updateDNSRecords struct {
	Domainname     string       `json:"domainname"`
	Customernumber int          `json:"customernumber"`
	Apikey         string       `json:"apikey"`
	Apisessionid   string       `json:"apisessionid"`
	Dnsrecordset   dnsRecordSet `json:"dnsrecordset"`
}

func callNCAPI(payload request) (*answer, error) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	response, err := netClient.Post("https://ccp.netcup.net/run/webservice/servers/endpoint.php?JSON", "application/json", bytes.NewBuffer(buf))

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	body, err := ioutil.ReadAll(response.Body)

	if err != nil {
		return nil, err
	}

	var answer answer

	err = json.Unmarshal(body, &answer)

	if err != nil {
		return nil, err
	}

	return &answer, nil
}

type configuration struct {
	Customernumber int    `json:"customernumber"`
	Apikey         string `json:"apikey"`
	Apipassword    string `json:"apipassword"`
}

func readConfiguration() (*configuration, error) {
	confFile, err := os.Open("/etc/nc-dyndns/config.json")
	if err != nil {
		confFile, err = os.Open("config.json")

		if err != nil {
			return nil, err
		}
	}

	confBytes, err := ioutil.ReadAll(confFile)

	confFile.Close()

	if err != nil {
		return nil, err
	}

	var configuration configuration
	err = json.Unmarshal(confBytes, &configuration)

	if err != nil {
		return nil, err
	}

	return &configuration, nil
}

func main() {
	configuration, err := readConfiguration()

	if err != nil {
		panic("Could not read configuration, aborting.")
	}

	var hostname string
	flag.StringVar(&hostname, "host", "", "Supply the hostname to set the A record for.")
	var domain string
	flag.StringVar(&domain, "domain", "", "Supply the zone name to set the A record in.")
	var ip string
	flag.StringVar(&ip, "ip", "", "Supply the IP address to set as destination.")

	flag.Parse()

	if hostname == "" || domain == "" || ip == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	answer, err := callNCAPI(request{"login", login{configuration.Customernumber, configuration.Apikey, configuration.Apipassword}})

	if err != nil {
		panic(err)
	}

	if answer.Status != "success" {
		panic(fmt.Sprintf("%+v\n", *answer))
	}

	session, err := answer.GetLoginResponse()

	if err != nil {
		panic(err)
	}

	answer, err = callNCAPI(request{"infoDnsRecords", infoDNSRecords{domain, configuration.Customernumber, configuration.Apikey, session.Apisessionid}})

	if err != nil {
		panic(err)
	}

	if answer.Status != "success" {
		panic(fmt.Sprintf("%+v\n", *answer))
	}

	zone, err := answer.GetDNSRecordResponse()

	if err != nil {
		panic(err)
	}

	if answer.Status != "success" {
		panic(fmt.Sprintf("%+v\n", *answer))
	}

	entry := zone.firstOrNewRecord(hostname, "A")

	if entry.Destination != ip {
		old := entry.Destination
		entry.Destination = ip
		answer, err = callNCAPI(request{"updateDnsRecords", updateDNSRecords{domain, configuration.Customernumber, configuration.Apikey, session.Apisessionid, dnsRecordSet{[]dnsRecord{*entry}}}})

		if err != nil {
			panic(err)
		}

		if answer.Status != "success" {
			panic(fmt.Sprintf("%+v\n", *answer))
		}

		fmt.Printf("IP Address changed from %v to %v\n", old, ip)
	} else {
		fmt.Println("IP Address did not change.")
	}

	answer, err = callNCAPI(request{"logout", logout{configuration.Customernumber, configuration.Apikey, session.Apisessionid}})

	if err != nil {
		panic(err)
	}

	if answer.Status != "success" {
		panic(fmt.Sprintf("%+v\n", *answer))
	}

}
