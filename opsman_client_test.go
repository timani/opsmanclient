package opsmanclient_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotalservices/opsmanclient"
)

var _ = Describe("OpsmanClient", func() {
	Describe("get api version", func() {
		var (
			err error
			ver string
		)
		Context("when opsman version is 2.0", func() {
			JustBeforeEach(func() {
				opsman.InitializeAPIVersionTest(opsmanclient.Version{Version: "2.0"}, false)
				ver, err = c.GetAPIVersion()
			})
			It("returns 2.0", func() {
				Expect(ver).To(Equal("2.0"))
			})
			It("does not error", func() {
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Context("when opsman version is NOT 2.0", func() {
			JustBeforeEach(func() {
				opsman.InitializeAPIVersionTest(opsmanclient.Version{Version: "3.0"}, false)
				ver, err = c.GetAPIVersion()
			})
			It("validate should fail", func() {
				err = opsmanclient.ValidateAPIVersion(ver)
				Expect(err).To(MatchError("This version of Ops Manager (using api version ''" + ver + "') is not supported"))
			})
		})
	})
})
