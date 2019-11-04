package plunder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/plunder-app/plunder/pkg/apiserver"
)

// DeleteMachine will remove a provisioned machine
func (c *Client) DeleteMachine(ipAddress string) error {

	destroyMap := destroyCommand(ipAddress)

	// Marshall the parlay submission (runs the set of destroy commands)
	b, err := json.Marshal(destroyMap)
	if err != nil {
		return err
	}

	// Set Parlay API path and POST
	ep, resp := apiserver.FindFunctionEndpoint(c.address, c.server, "parlay", http.MethodPost)
	if resp.Error != "" || resp.FriendlyError != "" {
		return fmt.Errorf(resp.FriendlyError)
	}

	c.address.Path = ep.Path
	response, err := apiserver.ParsePlunderPost(c.address, c.server, b)
	if err != nil {

		return fmt.Errorf(response.Error)
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		// TODO - if this error occurs it's because the machine doesn't exist
		//plunderMachine.Finalizers = util.Filter(plunderMachine.Finalizers, infrav1.MachineFinalizer)
		//logger.Info(fmt.Sprintf("Removing Machine with address [%s] from config, it may need removing manually", plunderM))
		return fmt.Errorf(resp.FriendlyError)

	}

	// Set Parlay API path and POST
	ep, resp = apiserver.FindFunctionEndpoint(c.address, c.server, "deploymentAddress", http.MethodDelete)
	if resp.Error != "" {
		return fmt.Errorf(resp.Error)

	}
	c.address.Path = ep.Path + "/" + strings.Replace(ipAddress, ".", "-", -1)
	response, err = apiserver.ParsePlunderDelete(c.address, c.server)
	if err != nil {
		return err
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return fmt.Errorf(resp.Error)

	}
	return nil
}
