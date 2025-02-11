package app

import (
	"net/http"

	"github.com/luthermonson/go-proxmox"
)

func NewClient(tokenID string, secret string) *proxmox.Client {
	HTTPClient := http.Client{}

	client := proxmox.NewClient("https://pve.tronnet.net/api2/json",
		proxmox.WithHTTPClient(&HTTPClient),
		proxmox.WithAPIToken(tokenID, secret),
	)

	return client
}
