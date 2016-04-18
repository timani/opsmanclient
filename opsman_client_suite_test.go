package opsmanclient_test

import (
	"fmt"
	"io/ioutil"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotalservices/opsmanclient"
	"github.com/pivotalservices/opsmanclient/mockopsman"

	"testing"
)

var (
	c          *opsmanclient.OpsManAPI
	opsman     *mockopsman.OpsManager
	shouldFail bool
)
var _ = BeforeEach(func() {
	opsman = mockopsman.New()
})

var _ = AfterEach(func() {
	opsman.Close()
})

var _ = JustBeforeEach(func() {
	c = opsmanclient.New(opsman.URL, "admin", "admin", "", false)
})

func TestOpsManClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "OpsManClient Suite")
}

func fixture(name string) string {
	filePath := path.Join("fixtures", name)
	contents, err := ioutil.ReadFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("could not read fixture: %s", name))
	}

	return string(contents)
}
