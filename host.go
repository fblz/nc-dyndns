package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net"
	"strings"

	"github.com/cloudflare/cloudflare-go"
)

type Host struct {
	Hostname string `json:"hostname"`
	Zone     string `json:"zone"`
	APIToken string `json:"api_token"`
	ZoneID   string `json:"zone_id,omitempty"`
	RecordID string `json:"record_id,omitempty"`
	IP       string `json:"-"`
	WebToken string `json:"web_token,omitempty"`
}

func (h *Host) FQDN() string {
	return h.Hostname + "." + h.Zone
}

func (h *Host) Populate() error {
	err := h.populateWebToken()
	if err != nil {
		return err
	}

	err = h.populateAPIInfo()
	if err != nil {
		return err
	}

	return nil
}

func (h *Host) populateAPIInfo() error {
	api, err := cloudflare.NewWithAPIToken(h.APIToken)
	if err != nil {
		return err
	}

	if strings.TrimSpace(h.ZoneID) == "" {
		h.ZoneID, err = api.ZoneIDByName(h.Zone)
		if err != nil {
			return err
		}
	}

	if strings.TrimSpace(h.RecordID) == "" {
		ctx := context.Background()
		var filter cloudflare.DNSRecord
		filter.Name = h.FQDN()
		filter.Type = "A"
		records, err := api.DNSRecords(ctx, h.ZoneID, filter)
		if err != nil {
			return err
		}

		var record cloudflare.DNSRecord

		if len(records) == 0 {
			result, err := h.createRecord()
			if err != nil {
				return err
			}
			record = *result
		} else {
			record = records[0]
		}

		h.RecordID = record.ID
		h.IP = record.Content
	}

	if strings.TrimSpace(h.IP) == "" {
		ctx := context.Background()
		record, err := api.DNSRecord(ctx, h.ZoneID, h.RecordID)
		if err != nil {
			return err
		}
		h.IP = record.Content
	}

	return nil
}

func (h *Host) populateWebToken() error {
	if strings.TrimSpace(h.WebToken) != "" {
		return nil
	}

	byteLength := 32
	random := make([]byte, byteLength)
	_, err := rand.Read(random)
	if err != nil {
		return err
	}

	h.WebToken = base64.RawURLEncoding.EncodeToString(random)
	return nil
}

func (h *Host) updateIP(ip net.IP) error {
	api, err := cloudflare.NewWithAPIToken(h.APIToken)
	if err != nil {
		return err
	}

	var record cloudflare.DNSRecord
	record.Content = ip.String()

	ctx := context.Background()
	err = api.UpdateDNSRecord(ctx, h.ZoneID, h.RecordID, record)
	if err != nil {
		return err
	}

	h.IP = ip.String()
	return nil
}

func (h *Host) createRecord() (*cloudflare.DNSRecord, error) {
	api, err := cloudflare.NewWithAPIToken(h.APIToken)
	if err != nil {
		return nil, err
	}

	var record cloudflare.DNSRecord
	record.Name = h.FQDN()
	record.Type = "A"
	record.Content = "0.0.0.0"
	record.TTL = 300
	proxy := false
	record.Proxied = &proxy

	result, err := api.CreateDNSRecord(context.Background(), h.ZoneID, record)
	if err != nil {
		return nil, err
	}

	return &result.Result, nil
}
