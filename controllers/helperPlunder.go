package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/services"
)

func findUnleasedServer(u *url.URL, c *http.Client) (foundMAC string, err error) {
	ep, resp := apiserver.FindFunctionEndpoint(u, c, "dhcp", http.MethodGet)
	if resp.Error != "" {
		return foundMAC, fmt.Errorf(resp.Error)

	}

	u.Path = path.Join(u.Path, ep.Path+"/unleased")

	response, err := apiserver.ParsePlunderGet(u, c)
	if err != nil {
		return foundMAC, fmt.Errorf(resp.Error)
	}
	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return foundMAC, fmt.Errorf(resp.Error)
	}
	var unleased []services.Lease

	err = json.Unmarshal(response.Payload, &unleased)
	if err != nil {
		return foundMAC, fmt.Errorf(resp.Error)
	}

	// Iterate through all known addresses and find a free one that looks "recent"
	for i := range unleased {
		if time.Since(unleased[i].Expiry).Minutes() < 10 {
			foundMAC = unleased[i].MAC
		}
	}
	return
}

func createDeployment(u *url.URL, c *http.Client, b []byte) error {
	ep, resp := apiserver.FindFunctionEndpoint(u, c, "deployment", http.MethodPost)
	if resp.Error != "" {
		return fmt.Errorf(resp.Error)

	}

	u.Path = ep.Path

	response, err := apiserver.ParsePlunderPost(u, c, b)
	if err != nil {
		return err
	}
	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return fmt.Errorf(resp.Error)
	}
	return nil
}
