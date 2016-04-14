package opsmanclient

import (
	"bytes"
	"encoding/json"
	"fmt"

	http "github.com/pivotalservices/opsmanclient/http"
)

// GetAPIVersion returns the Ops Man API version
func (c *Client) GetAPIVersion() (string, error) {
	resp, err := http.SendRequest("GET", fmt.Sprintf("%s/api/api_version", c.opsmanURL), c.opsmanUsername, c.opsmanPassword, "")
	if err != nil {
		return "", err
	}
	res := bytes.NewBufferString(resp)
	decoder := json.NewDecoder(res)
	var ver Version
	err = decoder.Decode(&ver)
	if err != nil {
		return "", err
	}

	return ver.Version, nil
}

// ValidateAPIVersion checks for a supported API version
func ValidateAPIVersion(version string) error {

	if version != "2.0" {
		return fmt.Errorf("This version of Ops Manager (using api version ''" + version + "') is not supported")
	}

	return nil
}
