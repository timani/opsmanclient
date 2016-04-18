package opsmanclient

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	nhttp "net/http"
	urllib "net/url"
	"os"

	"github.com/op/go-logging"
	"github.com/pivotalservices/gtils/command"
	ghttp "github.com/pivotalservices/gtils/http"
	"github.com/pivotalservices/gtils/uaa"
	"github.com/pivotalservices/opsmanclient/http"
)

// OpsManAPI implements the Ops Manager API
type OpsManAPI struct {
	opsmanURL         string
	opsmanUsername    string
	opsmanPassword    string
	opsmanPassphrase  string
	HTTPClient        HTTPClient
	logger            *logging.Logger
	AssetsUploader    httpUploader
	SettingsRequestor httpRequestor
}

type httpUploader func(conn ghttp.ConnAuth, paramName, filename string, fileSize int64, fileRef io.Reader, params map[string]string) (*nhttp.Response, error)

type httpRequestor interface {
	Get(ghttp.HttpRequestEntity) ghttp.RequestAdaptor
	Post(ghttp.HttpRequestEntity, io.Reader) ghttp.RequestAdaptor
	Put(ghttp.HttpRequestEntity, io.Reader) ghttp.RequestAdaptor
}

// HTTPClient is the interface for making HTTP calls to Ops Man API
type HTTPClient interface {
	Get(url string) (resp *nhttp.Response, err error)
	Post(url string, bodyType string, body io.Reader) (resp *nhttp.Response, err error)
}

// New creates a Client for calling Ops Man API
func New(opsmanURL, opsmanUsername, opsmanPassword, opsmanPassphrase string, isS3 bool) *OpsManAPI {
	logger := setupLogger()
	return &OpsManAPI{
		opsmanURL: opsmanURL,
		HTTPClient: http.New(http.Config{
			NoFollowRedirect:                  false,
			DisableTLSCertificateVerification: true, // TODO: Let user decide
			Username: opsmanUsername,
			Password: opsmanPassword,
		}),
		logger:            logger,
		opsmanUsername:    opsmanUsername,
		opsmanPassword:    opsmanPassword,
		opsmanPassphrase:  opsmanPassphrase,
		AssetsUploader:    httpUploader(getUploader(isS3)),
		SettingsRequestor: ghttp.NewHttpGateway(),
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

// GetInstallationSettingsRaw returns installation settings in raw format
func (c *OpsManAPI) GetInstallationSettingsRaw() ([]byte, error) {
	resp, err := c.HTTPClient.Get(fmt.Sprintf("%s/api/installation_settings", c.opsmanURL))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == nhttp.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != nhttp.StatusOK {
		return nil, unexpectedStatusErr(resp, nhttp.StatusOK)
	}

	respJSON := make(map[string]string)
	if err := json.NewDecoder(resp.Body).Decode(&respJSON); err != nil {
		return nil, fmt.Errorf("error unmarshalling GetInstallationSettings json response: %s", err)
	}
	fmt.Printf("respJSON: %+v", respJSON)
	return []byte(respJSON["installation_settings"]), nil
}

// GetInstallationSettingsBuffered retrieves all the installation settings from OpsMan
// and returns them in a buffered reader
func (c *OpsManAPI) GetInstallationSettingsBuffered() (io.Reader, error) {

	var bytesBuffer = new(bytes.Buffer)
	url := fmt.Sprintf("%s/api/installation_settings", c.opsmanURL)
	c.logger.Debug(fmt.Sprintf("Exporting url '%s'", url))

	if err := c.saveHTTPResponse(url, bytesBuffer); err != nil {
		return nil, err
	}
	return bytesBuffer, nil
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

//NewSSHExecuter creates an ssh executer for running commands not available via http
func NewSSHExecuter(username, password, host, sshKey string, sshPort int) (command.Executer, error) {
	return command.NewRemoteExecutor(command.SshConfig{
		Username: username,
		Password: password,
		Host:     host,
		Port:     sshPort,
		SSLKey:   sshKey,
	})
}

func (c *OpsManAPI) SaveDeployments(e command.Executer, bw io.WriteCloser) error {
	cmd := "cd /var/tempest/workspaces/default && tar cz deployments"
	return e.Execute(bw, cmd)
}

func (c *OpsManAPI) SaveInstallation(backupWriter io.WriteCloser) error {
	err := c.exportFile("%s/api/installation_settings", "installation.json", backupWriter)
	if err != nil {
		return err
	}
	return c.exportFile("%s/api/installation_asset_collection", "installation.zip", backupWriter)
}

func (c *OpsManAPI) exportFile(urlFormat string, filename string, backupWriter io.WriteCloser) error {
	url := fmt.Sprintf(urlFormat, c.opsmanURL)

	c.logger.Debugf("Exporting file url:%s, filename: %s", url, filename)

	return c.saveHTTPResponse(url, backupWriter)
}

func (c *OpsManAPI) saveHTTPResponse(url string, dest io.Writer) error {
	var resp *nhttp.Response
	var err error
	c.logger.Debug("attempting to auth against", url)

	if resp, err = c.oauthHTTPGet(url); err != nil {
		c.logger.Infof("falling back to basic auth for legacy system", err)
		resp, err = c.legacyHTTPGet(url)
	}

	if err == nil && resp.StatusCode == nhttp.StatusOK {
		defer resp.Body.Close()
		_, err = io.Copy(dest, resp.Body)

	} else if resp != nil && resp.StatusCode != nhttp.StatusOK {
		errMsg, _ := ioutil.ReadAll(resp.Body)
		err = errors.New(string(errMsg[:]))
	}

	if err != nil {
		return fmt.Errorf("error in save http request, %v", err)
	}
	return nil
}

func (c *OpsManAPI) legacyHTTPGet(url string) (*nhttp.Response, error) {
	resp, err := c.SettingsRequestor.Get(ghttp.HttpRequestEntity{
		Url:         url,
		Username:    c.opsmanUsername,
		Password:    c.opsmanPassword,
		ContentType: "application/octet-stream",
	})()
	c.logger.Debugf("called basic auth on legacy ops manager", url, err)
	return resp, err
}

func (c *OpsManAPI) oauthHTTPGet(urlString string) (*nhttp.Response, error) {
	var uaaURL, _ = urllib.Parse(urlString)
	var opsManagerUsername = c.opsmanUsername
	var opsManagerPassword = c.opsmanPassword
	var clientID = "opsman"
	var clientSecret = ""
	var response *nhttp.Response
	var token string
	var err error
	c.logger.Debug("aquiring your token from: ", uaaURL, urlString)

	if token, err = uaa.GetToken("https://"+uaaURL.Host+"/uaa", opsManagerUsername, opsManagerPassword, clientID, clientSecret); err == nil {
		c.logger.Debug("your token", token, "https://"+uaaURL.Host+"/uaa")
		requestor := c.SettingsRequestor
		response, err = requestor.Get(ghttp.HttpRequestEntity{
			Url:           urlString,
			ContentType:   "application/octet-stream",
			Authorization: "Bearer " + token,
		})()
	}
	return response, err
}

func (c *OpsManAPI) ImportInstallation(e command.Executer, backupDir string, backupReader io.ReadCloser, removeBoshManifest bool) error {
	var err error
	defer func() {
		if err == nil && removeBoshManifest {
			c.logger.Debug("removing deployment files")
			err = c.removeExistingDeploymentFiles(e)
		}
	}()
	installAssetsURL := fmt.Sprintf("%s/api/installation_asset_collection", c.opsmanURL)
	c.logger.Debug("uploading installation assets installAssetsURL: %s", installAssetsURL)
	return c.importInstallationPart(installAssetsURL, "installation.zip", "installation[file]", backupDir, backupReader)
}

func (c *OpsManAPI) importInstallationPart(url, filename, fieldname, filePath string, backupReader io.ReadCloser) error {
	var err error

	var resp *nhttp.Response
	conn := ghttp.ConnAuth{
		Url:      url,
		Username: c.opsmanUsername,
		Password: c.opsmanPassword,
	}
	// filePath := path.Join(c.BackupContext.TargetDir, c.BackupDir, filename)
	bufferedReader := bufio.NewReader(backupReader)
	c.logger.Debugf("upload request, fieldname: %s, filePath: %s, conn: %s", fieldname, filePath, conn)
	creds := map[string]string{
		"password":   c.opsmanPassword,
		"passphrase": c.opsmanPassphrase,
	}
	resp, err = c.AssetsUploader(conn, fieldname, filePath, -1, bufferedReader, creds)

	if err == nil && resp.StatusCode == nhttp.StatusOK {
		c.logger.Debug("request for %s succeeded with status: %s", url, resp.Status)

	} else if resp != nil && resp.StatusCode != nhttp.StatusOK {
		return fmt.Errorf("request for %s failed with status: %s", url, resp.Status)
	}

	if err != nil {
		return fmt.Errorf("error uploading installation, %v", err)
	}
	return err
}

func (c *OpsManAPI) removeExistingDeploymentFiles(e command.Executer) error {
	var w bytes.Buffer
	deploymentsFile := "/var/tempest/workspaces/default/deployments/bosh-deployments.yml"
	command := fmt.Sprintf("if [ -f %s ]; then sudo rm %s;fi", deploymentsFile, deploymentsFile)
	return e.Execute(&w, command)
}

func unexpectedStatusErr(response *nhttp.Response, expectedStatus int) error {
	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	var bodyStr string
	if err == nil {
		bodyStr = string(body)
	} else {
		bodyStr = "COULDN'T READ RESPONSE BODY"
	}

	return fmt.Errorf("expected status %d, was %d. Response Body: %s", expectedStatus, response.StatusCode, bodyStr)
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

func getUploader(isS3 bool) httpUploader {
	uploader := ghttp.LargeMultiPartUpload

	if isS3 {
		uploader = ghttp.MultiPartUpload
	}
	return uploader
}

func setupLogger() *logging.Logger {
	if logLevel, err := logging.LogLevel(os.Getenv("LOG_LEVEL")); err == nil {
		logging.SetLevel(logLevel, "opsmanclient")
	} else {
		logging.SetLevel(logging.INFO, "opsmanclient")
	}
	logging.SetFormatter(logging.GlogFormatter)
	return logging.MustGetLogger("opsmanclient")
}
