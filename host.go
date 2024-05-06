package main

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
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
		records, _, err := api.ListDNSRecords(context.Background(), cloudflare.ZoneIdentifier(h.ZoneID), cloudflare.ListDNSRecordsParams{Name: h.FQDN(), Type: "A"})
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
		record, err := api.GetDNSRecord(context.Background(), cloudflare.ZoneIdentifier(h.ZoneID), h.RecordID)
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

	record, err := api.UpdateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(h.ZoneID), cloudflare.UpdateDNSRecordParams{ID: h.RecordID, Content: ip.String()})
	if err != nil {
		return err
	}

	if record.Content != ip.String() {
		return fmt.Errorf("After UpdateDNSRecord, Cloudflare API did not reflect the IP changes on host " + h.Hostname)
	}

	h.IP = record.Content
	return nil
}

func (h *Host) createRecord() (*cloudflare.DNSRecord, error) {
	api, err := cloudflare.NewWithAPIToken(h.APIToken)
	if err != nil {
		return nil, err
	}

	// Context for why this pointer to bool is required
	// https://github.com/cloudflare/cloudflare-go/issues/568
	proxy := false
	result, err := api.CreateDNSRecord(context.Background(), cloudflare.ZoneIdentifier(h.ZoneID), cloudflare.CreateDNSRecordParams{Name: h.FQDN(), Type: "A", Content: "0.0.0.0", TTL: 300, Proxied: &proxy})
	if err != nil {
		return nil, err
	}

	return &result, nil
}
