package netcupapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
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

type host struct {
	Hostname string `json:"hostname"`
	Domain   string `json:"domain"`
	Token    string `json:"token"`
	IP       string `json:"-"`
}

func (h *host) Key() string {
	return h.Hostname + "." + h.Domain
}

// NetcupAPI provides a incomplete netcup API to set A names
type NetcupAPI struct {
	Customernumber  int    `json:"customernumber"`
	Apikey          string `json:"apikey"`
	Apipassword     string `json:"apipassword"`
	AuthorizedHosts []host `json:"authorized_hosts"`
}

// New reads the configuration file and creates a API object
func New(configPath string) (*NetcupAPI, error) {
	confFile, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}

	confBytes, err := ioutil.ReadAll(confFile)

	confFile.Close()

	if err != nil {
		return nil, err
	}

	this := new(NetcupAPI)
	err = json.Unmarshal(confBytes, this)

	if err != nil {
		return nil, err
	}

	return this, nil
}

func (nc *NetcupAPI) updateDNSHostname(host *host, ip net.IP) error {
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

	answer, err = callNCAPI(request{"infoDnsRecords", infoDNSRecords{host.Domain, nc.Customernumber, nc.Apikey, session.Apisessionid}})

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

	entry := zone.firstOrNewRecord(host.Hostname, "A")

	entry.Destination = ip.String()
	answer, err = callNCAPI(request{"updateDnsRecords", updateDNSRecords{host.Domain, nc.Customernumber, nc.Apikey, session.Apisessionid, dnsRecordSet{[]dnsRecord{*entry}}}})

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

func (nc *NetcupAPI) getHost(fqdn string) (*host, error) {
	for i := range nc.AuthorizedHosts {
		if nc.AuthorizedHosts[i].Key() == fqdn {
			return &nc.AuthorizedHosts[i], nil
		}
	}

	return nil, fmt.Errorf("Unknown fqdn %s", fqdn)
}

func validateQuery(query url.Values) (map[string]string, error) {
	params := []string{"fqdn", "ipv4", "token"}
	validQuery := make(map[string]string)

	for _, param := range params {
		queryItems, ok := query[param]

		if !ok || len(queryItems) < 1 {
			return nil, fmt.Errorf("Parameter %s is missing", param)
		}

		validQuery[param] = queryItems[0]
	}

	return validQuery, nil
}

// DynDNSHandler provides a handler function for the web package
func (nc *NetcupAPI) DynDNSHandler(w http.ResponseWriter, r *http.Request) {
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
	host, err := nc.getHost(query["fqdn"])

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Not authorized to set this fqdn", http.StatusForbidden)
		return
	}

	//validate token by searching in the config file
	if host.Token != query["token"] {
		fmt.Printf("Invalid token supplied for fqdn %s\n", host.Key())
		http.Error(w, "Not authorized to set this fqdn", http.StatusForbidden)
		return
	}

	if host.IP == ip.String() {
		fmt.Println("IP did not change")
		fmt.Fprint(w, "success")
		return
	}

	fmt.Printf("Changing IP Address from %v to %v\n", host.IP, ip.String())

	interval := 200

	for index := 0; index < 5; index++ {
		err = nc.updateDNSHostname(host, ip)
		_, ok := err.(net.Error)

		if !ok {
			break
		}

		fmt.Printf("Encountered network error. Retrying in %d seconds.\n%v\n", interval, err)

		time.Sleep(time.Duration(interval) * time.Millisecond)
		interval += interval
	}

	if err != nil {
		fmt.Println(err)
		http.Error(w, "Something unexpected happened", http.StatusInternalServerError)
		return
	}

	host.IP = ip.String()

	fmt.Println("Success")
	fmt.Fprint(w, "success")
}
