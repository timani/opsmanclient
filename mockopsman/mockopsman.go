package mockopsman

import (
	"net/http/httptest"
	"sync"

	"github.com/gorilla/mux"
	"github.com/pivotalservices/opsmanclient"
)

type OpsManager struct {
	// GetAPIVersion
	APIVersion             string
	GetAPIVersionCalls     int
	stubbedAPIVersionCalls []StubbedAPIVersionCall

	// common
	shouldFail bool
	FailBody   string

	*httptest.Server
	*sync.Mutex
}

type StubbedAPIVersionCall struct {
	ExpectedVersion opsmanclient.Version
	ShouldFail      bool
}

func New() *OpsManager {
	om := &OpsManager{Mutex: new(sync.Mutex)}
	router := mux.NewRouter()
	router.HandleFunc("/api/api_version", om.getAPIVersion).Methods("GET")
	om.Server = httptest.NewServer(router)
	om.FailBody = "epic fail"
	return om
}
