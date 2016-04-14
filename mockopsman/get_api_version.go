package mockopsman

import (
	"encoding/json"
	"fmt"
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotalservices/opsmanclient"
)

func (o *OpsManager) InitializeAPIVersionTest(expectedVersion opsmanclient.Version, shouldFail bool) {
	o.shouldFail = shouldFail // TODO: See usages, and try to remove
	o.InitializeAPIVersionsTest([]StubbedAPIVersionCall{
		{
			ExpectedVersion: expectedVersion,
			ShouldFail:      shouldFail,
		},
	})
}

func (o *OpsManager) InitializeAPIVersionsTest(calls []StubbedAPIVersionCall) {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()

	o.stubbedAPIVersionCalls = calls
	o.GetAPIVersionCalls = 0
}

func (o *OpsManager) getAPIVersion(w http.ResponseWriter, r *http.Request) {
	o.Mutex.Lock()
	defer o.Mutex.Unlock()

	defer GinkgoRecover()
	if o.GetAPIVersionCalls >= len(o.stubbedAPIVersionCalls) {
		Fail(fmt.Sprintf("unstubbed call to api_version call received, current stubs %v, getVersion haved called %d times before", o.stubbedAPIVersionCalls, o.GetAPIVersionCalls))
	}

	currentStub := o.stubbedAPIVersionCalls[o.GetAPIVersionCalls]

	responseBytes, err := json.Marshal(currentStub.ExpectedVersion)
	Expect(err).NotTo(HaveOccurred())

	o.GetAPIVersionCalls++

	if currentStub.ShouldFail {
		w.WriteHeader(500)
		return
	}

	w.Write(responseBytes)
}
