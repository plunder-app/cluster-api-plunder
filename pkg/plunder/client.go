package plunder

import (
	"net/http"
	"net/url"

	"github.com/plunder-app/plunder/pkg/apiserver"
	"github.com/plunder-app/plunder/pkg/parlay/parlaytypes"
)

// Client defines all the components needed to interact with Plunder
type Client struct {
	address       *url.URL
	server        *http.Client
	deploymentMap *parlaytypes.TreasureMap
}

// NewClient -  a  this will attempt to create a new client for interacting with Plunder
func NewClient() (*Client, error) {
	// Find a machine for provisioning

	// TODO - Make config path configurable.
	u, c, err := apiserver.BuildEnvironmentFromConfig("plunderclient.yaml", "")
	if err != nil {
		return nil, err
	}
	return &Client{
		address: u,
		server:  c,
	}, nil
}
