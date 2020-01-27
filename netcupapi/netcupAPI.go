package netcupapi

import (
	"bytes"
	"encoding/json"
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

type netcupAPI struct {
	Customernumber int      `json:"customernumber"`
	Apikey         string   `json:"apikey"`
	Apipassword    string   `json:"apipassword"`
	AllowedDomains []string `json:"allowed_domains"`
	AllowedHosts   []string `json:"allowed_hosts"`
}

func New(configPath string) (*netcupAPI, error) {
	confFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	confBytes, err := ioutil.ReadAll(confFile)

	confFile.Close()

	if err != nil {
		return nil, err
	}

	this := new(netcupAPI)
	err = json.Unmarshal(confBytes, this)

	if err != nil {
		return nil, err
	}

	return this, nil
}

func (nc *netcupAPI) updateDNSHostname(hostname string, domain string, ip string) error {
	answer, err := callNCAPI(request{"login", login{nc.Customernumber, nc.Apikey, nc.Apipassword}})

	if err != nil {
		return err
	}

	if answer.Status != "success" {
		return fmt.Errorf("%+v", *answer)
	}

	session, err := answer.GetLoginResponse()

	if err != nil {
		return err
	}

	answer, err = callNCAPI(request{"infoDnsRecords", infoDNSRecords{domain, nc.Customernumber, nc.Apikey, session.Apisessionid}})

	if err != nil {
		return err
	}

	if answer.Status != "success" {
		return fmt.Errorf("%+v", *answer)
	}

	zone, err := answer.GetDNSRecordResponse()

	if err != nil {
		return err
	}

	if answer.Status != "success" {
		return fmt.Errorf("%+v", *answer)
	}

	entry := zone.firstOrNewRecord(hostname, "A")
	entry.Destination = ip
	answer, err = callNCAPI(request{"updateDnsRecords", updateDNSRecords{domain, nc.Customernumber, nc.Apikey, session.Apisessionid, dnsRecordSet{[]dnsRecord{*entry}}}})

	if err != nil {
		return err
	}

	if answer.Status != "success" {
		return fmt.Errorf("%+v", *answer)
	}

	answer, err = callNCAPI(request{"logout", logout{nc.Customernumber, nc.Apikey, session.Apisessionid}})

	if err != nil {
		return err
	}

	if answer.Status != "success" {
		return fmt.Errorf("%+v", *answer)
	}

	return nil
}

func (nc *netcupAPI) domainAllowed(domain string) bool {
	for i := range nc.AllowedDomains {
		if nc.AllowedDomains[i] == domain {
			return true
		}
	}
	return false
}

func (nc *netcupAPI) hostnameAllowed(hostname string) bool {
	for i := range nc.AllowedHosts {
		if nc.AllowedHosts[i] == hostname {
			return true
		}
	}
	return false
}

func (nc *netcupAPI) DynDNSHandler(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	hostname, ok := query["hostname"]

	if !ok || len(hostname) < 1 {
		msg := "Parameter hostname is missing."
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	domain, ok := query["domain"]

	if !ok || len(domain) < 1 {
		msg := "Parameter domain is missing."
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	ip, ok := query["ip"]

	if !ok || len(ip) < 1 {
		msg := "Parameter ip is missing."
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	if !nc.domainAllowed(domain[0]) {
		msg := fmt.Sprintf("Domain %s is not allowed.", domain[0])
		fmt.Println(msg)
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	if !nc.hostnameAllowed(hostname[0]) {
		msg := fmt.Sprintf("Hostname %s not allowed.", hostname[0])
		fmt.Println(msg)
		http.Error(w, msg, http.StatusForbidden)
		return
	}

	err := nc.updateDNSHostname(hostname[0], domain[0], ip[0])

	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	msg := fmt.Sprintf("Updated IP Address to %s", ip[0])
	fmt.Println(msg)
	fmt.Fprint(w, msg)
}
