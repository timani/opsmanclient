package opsmanclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	nhttp "net/http"

	"github.com/pivotalservices/opsmanclient/http"
)

// OpsManAPI implements the Ops Manager API
type OpsManAPI struct {
	opsmanURL  string
	HTTPClient HTTPClient
}

// HTTPClient is the interface for making HTTP calls to Ops Man API
type HTTPClient interface {
	Get(url string) (resp *nhttp.Response, err error)
	Post(url string, bodyType string, body io.Reader) (resp *nhttp.Response, err error)
}

// New creates a Client for calling Ops Man API
func New(opsmanURL, opsmanUsername, opsmanPassword string) *OpsManAPI {
	return &OpsManAPI{
		opsmanURL: opsmanURL,
		HTTPClient: http.New(http.Config{
			NoFollowRedirect:                  false,
			DisableTLSCertificateVerification: true, // TODO: Let user decide
			Username: opsmanUsername,
			Password: opsmanPassword,
		}),
	}
}

// GetCFDeployment returns the Elastic-Runtime deployment created by your Ops Manager
func (c *OpsManAPI) GetCFDeployment(installation *InstallationSettings, products []Products) (*Deployment, error) {
	cfRelease := getProductGUID(products, "cf")
	if cfRelease == "" {
		return nil, fmt.Errorf("cf release not found")
	}

	return NewDeployment(installation, cfRelease), nil
}

// GetInstallationSettings retrieves installation settings for cf deployment
func (c *OpsManAPI) GetInstallationSettings() (*InstallationSettings, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/installation_settings", c.opsmanURL))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := bytes.NewBufferString(string(body))
	decoder := json.NewDecoder(res)
	var installation *InstallationSettings
	err = decoder.Decode(&installation)

	return installation, err
}

// GetProducts returns all the products in an OpsMan installation
func (c *OpsManAPI) GetProducts() ([]Products, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/installation_settings/products", c.opsmanURL))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	res := bytes.NewBufferString(string(body))
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
