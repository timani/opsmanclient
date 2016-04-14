package opsmanclient

import (
	"bytes"
	"encoding/json"
	"fmt"

	http "github.com/pivotalservices/opsmanclient/http"
)

// Client - Ops Manager API client
type Client struct {
	opsmanURL      string
	opsmanUsername string
	opsmanPassword string
}

// New creates a Client for calling Ops Man API
func New(opsmanURL, opsmanUsername, opsmanPassword string) *Client {
	return &Client{
		opsmanURL:      opsmanURL,
		opsmanUsername: opsmanUsername,
		opsmanPassword: opsmanPassword,
	}
}

// GetCFDeployment returns the Elastic-Runtime deployment created by your Ops Manager
func (c *Client) GetCFDeployment(installation *InstallationSettings, products []Products) (*Deployment, error) {
	cfRelease := getProductGUID(products, "cf")
	if cfRelease == "" {
		return nil, fmt.Errorf("cf release not found")
	}

	return NewDeployment(installation, cfRelease), nil
}

// GetInstallationSettings retrieves installation settings for cf deployment
func (c *Client) GetInstallationSettings() (*InstallationSettings, error) {
	resp, err := http.SendRequest("GET", fmt.Sprintf("%s/api/installation_settings", c.opsmanURL), c.opsmanUsername, c.opsmanPassword, "")

	res := bytes.NewBufferString(resp)
	decoder := json.NewDecoder(res)
	var installation *InstallationSettings
	err = decoder.Decode(&installation)

	return installation, err
}

// GetProducts returns all the products in an OpsMan installation
func (c *Client) GetProducts() ([]Products, error) {
	resp, err := http.SendRequest("GET", fmt.Sprintf("%s/api/installation_settings/products", c.opsmanURL), c.opsmanUsername, c.opsmanPassword, "")
	if err != nil {
		return nil, err
	}
	res := bytes.NewBufferString(resp)
	decoder := json.NewDecoder(res)
	var products []Products
	err = decoder.Decode(&products)

	return products, err
}

// gets the product GUID for a given product type
func getProductGUID(products []Products, productType string) string {
	for prod := range products {
		if products[prod].Type == productType {
			return products[prod].GUID
		}
	}
	return ""
}
