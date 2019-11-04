package plunder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"time"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/services"
)

// FindMachine - will consult the plunder API to find a free machine
func (c *Client) FindMachine() (macAddress string, err error) {
	ep, resp := apiserver.FindFunctionEndpoint(c.address, c.server, "dhcp", http.MethodGet)
	if resp.Error != "" {
		return macAddress, fmt.Errorf(resp.Error)

	}

	c.address.Path = path.Join(c.address.Path, ep.Path+"/unleased")

	response, err := apiserver.ParsePlunderGet(c.address, c.server)
	if err != nil {
		return
	}
	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return macAddress, fmt.Errorf(resp.Error)
	}
	var unleased []services.Lease

	err = json.Unmarshal(response.Payload, &unleased)
	if err != nil {
		return
	}

	// Iterate through all known addresses and find a free one that looks "recent"
	for i := range unleased {
		if time.Since(unleased[i].Expiry).Minutes() < 10 {
			macAddress = unleased[i].Nic
		}
	}

	if macAddress == "" {
		err = fmt.Errorf("No available hardware for provisioning")
	}
	return
}
