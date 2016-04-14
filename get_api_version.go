package opsmanclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// GetAPIVersion returns the Ops Man API version
func (c *OpsManAPI) GetAPIVersion() (string, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/api_version", c.opsmanURL))
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	res := bytes.NewBufferString(string(body))
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
