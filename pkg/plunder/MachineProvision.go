package plunder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/plunderlogging"
	"github.com/plunder-app/plunder/pkg/services"
)

// ProvisionMachine - will provision a new machine
func (c *Client) ProvisionMachine(hostname, macAddress, ipAddress, deploymenType string) (err error) {

	// define the deployment configuration options
	d := services.DeploymentConfig{
		ConfigName: deploymenType,
		MAC:        macAddress,
		ConfigHost: services.HostConfig{
			IPAddress:  ipAddress,
			ServerName: hostname,
		},
	}

	ep, resp := apiserver.FindFunctionEndpoint(c.address, c.server, "deployment", http.MethodPost)
	if resp.Error != "" {
		return fmt.Errorf(resp.Error)

	}

	c.address.Path = ep.Path

	b, err := json.Marshal(d)
	if err != nil {
		return err
	}
	response, err := apiserver.ParsePlunderPost(c.address, c.server, b)
	if err != nil {
		return err
	}
	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return fmt.Errorf(resp.Error)

	}
	return nil
}

// ProvisionMachineWait - This will watch the provisioning process
func (c *Client) ProvisionMachineWait(ipAddress string) (result *string, err error) {

	uptimeMap := uptimeCommand(ipAddress)

	// Marshall the parlay submission (runs the uptime command)
	b, err := json.Marshal(uptimeMap)
	if err != nil {
		return
	}

	// Create the string that will be used to get the logs
	dashAddress := strings.Replace(ipAddress, ".", "-", -1)

	// Get the time
	t := time.Now()

	for {
		// Set Parlay API path and POST
		ep, resp := apiserver.FindFunctionEndpoint(c.address, c.server, "parlay", http.MethodPost)
		if resp.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}
		c.address.Path = ep.Path

		response, err := apiserver.ParsePlunderPost(c.address, c.server, b)
		if err != nil {
			return nil, err
		}

		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}

		// Sleep for five seconds
		time.Sleep(5 * time.Second)

		// Set the parlay API get logs path and GET
		ep, resp = apiserver.FindFunctionEndpoint(c.address, c.server, "parlayLog", http.MethodGet)
		if resp.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}
		c.address.Path = ep.Path + "/" + dashAddress

		response, err = apiserver.ParsePlunderGet(c.address, c.server)
		if err != nil {
			return nil, err
		}
		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}

		var logs plunderlogging.JSONLog

		err = json.Unmarshal(response.Payload, &logs)
		if err != nil {
			return nil, err
		}

		if logs.State == "Completed" {
			provisioningResult := fmt.Sprintf("Host has been succesfully provisioned OS in %s Seconds\n", time.Since(t).Round(time.Second))
			//r.Recorder.Eventf(plunderMachine, corev1.EventTypeNormal, "PlunderProvision", provisioningResult)

			return &provisioningResult, nil
		}
	}
}

// ProvisionKubernetes = will handle all of the tasks associated with deploying Kubernetes
func (c *Client) ProvisionKubernetes() (result *string, err error) {

	// Marshall the parlay submission (runs the uptime command)
	b, err := json.Marshal(c.deploymentMap)
	if err != nil {
		return
	}

	// Get the time
	t := time.Now()
	// Set Parlay API path and POST
	ep, resp := apiserver.FindFunctionEndpoint(c.address, c.server, "parlay", http.MethodPost)
	if resp.Error != "" {
		return nil, fmt.Errorf(resp.Error)

	}
	c.address.Path = ep.Path

	response, err := apiserver.ParsePlunderPost(c.address, c.server, b)
	if err != nil {
		return nil, err
	}

	// If an error has been returned then handle the error gracefully and terminate
	if response.FriendlyError != "" || response.Error != "" {
		return result, fmt.Errorf(resp.Error)

	}
	// Create the string that will be used to get the logs
	dashAddress := strings.Replace(c.deploymentMap.Deployments[0].Hosts[0], ".", "-", -1)

	for {
		// Sleep for five seconds
		time.Sleep(5 * time.Second)
		// Set the parlay API get logs path and GET
		ep, resp = apiserver.FindFunctionEndpoint(c.address, c.server, "parlayLog", http.MethodGet)
		if resp.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}
		c.address.Path = ep.Path + "/" + dashAddress

		response, err = apiserver.ParsePlunderGet(c.address, c.server)
		if err != nil {
			return nil, err
		}
		// If an error has been returned then handle the error gracefully and terminate
		if response.FriendlyError != "" || response.Error != "" {
			return nil, fmt.Errorf(resp.Error)

		}

		var logs plunderlogging.JSONLog

		err = json.Unmarshal(response.Payload, &logs)
		if err != nil {
			return result, err
		}

		if logs.State == "Completed" {
			// Report completion message
			provisioningResult := fmt.Sprintf("Task has been succesfully completed in %s Seconds\n", time.Since(t).Round(time.Second))
			result = &provisioningResult
			break
		} else if logs.State == "Failed" {
			// Report error message
			provisioningResult := fmt.Sprintf("Task has been failed after in %s Seconds\n", time.Since(t).Round(time.Second))
			result = &provisioningResult
			break
		}
	}
	return
}
