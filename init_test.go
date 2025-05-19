package acceptance_test

import (
	"fmt"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/onsi/gomega/format"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var stack struct {
	BuildArchive string
	RunArchive   string
	BuildImageID string
	RunImageID   string
}
var localRegistryPort int

func by(_ string, f func()) { f() }

func getFreePort() (port int, err error) {
	if l, err := net.Listen("tcp", ":0"); err == nil {
		defer l.Close()
		return l.Addr().(*net.TCPAddr).Port, nil
	}
	return 0, err
}

func TestAcceptance(t *testing.T) {
	format.MaxLength = 0
	SetDefaultEventuallyTimeout(30 * time.Second)

	Expect := NewWithT(t).Expect

	root, err := filepath.Abs(".")
	Expect(err).ToNot(HaveOccurred())

	localRegistryPort, err = getFreePort()
	Expect(err).ToNot(HaveOccurred())

	stack.BuildArchive = filepath.Join(root, "build", "build.oci")
	stack.BuildImageID = fmt.Sprintf("localhost:%d/stack-build-%s", localRegistryPort, uuid.NewString())

	stack.RunArchive = filepath.Join(root, "build", "run.oci")
	stack.RunImageID = fmt.Sprintf("localhost:%d/stack-run-%s", localRegistryPort, uuid.NewString())

	suite := spec.New("Acceptance", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Metadata", testMetadata)
	suite("BuildpackIntegration", testBuildpackIntegration)

	suite.Run(t)
}
