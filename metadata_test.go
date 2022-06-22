package acceptance_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	v1 "github.com/google/go-containerregistry/pkg/v1"
	"github.com/google/go-containerregistry/pkg/v1/layout"
	. "github.com/onsi/gomega"
	"github.com/paketo-buildpacks/packit/v2/vacation"
	"github.com/sclevine/spec"

	. "github.com/paketo-buildpacks/jam/integration/matchers"
	. "github.com/paketo-buildpacks/packit/v2/matchers"
)

func testMetadata(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		tmpDir string
	)

	it.Before(func() {
		var err error
		tmpDir, err = os.MkdirTemp("", "")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpDir)).To(Succeed())
	})

	it("builds full stack", func() {
		var buildReleaseDate, runReleaseDate time.Time

		by("confirming that the build image is correct", func() {
			dir := filepath.Join(tmpDir, "build-index")
			err := os.Mkdir(dir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			archive, err := os.Open(stack.BuildArchive)
			Expect(err).NotTo(HaveOccurred())
			defer archive.Close()

			err = vacation.NewArchive(archive).Decompress(dir)
			Expect(err).NotTo(HaveOccurred())

			path, err := layout.FromPath(dir)
			Expect(err).NotTo(HaveOccurred())

			index, err := path.ImageIndex()
			Expect(err).NotTo(HaveOccurred())

			indexManifest, err := index.IndexManifest()
			Expect(err).NotTo(HaveOccurred())

			Expect(indexManifest.Manifests).To(HaveLen(1))
			Expect(indexManifest.Manifests[0].Platform).To(Equal(&v1.Platform{
				OS:           "linux",
				Architecture: "amd64",
			}))

			image, err := index.Image(indexManifest.Manifests[0].Digest)
			Expect(err).NotTo(HaveOccurred())

			file, err := image.ConfigFile()
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Config.Labels).To(SatisfyAll(
				HaveKeyWithValue("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy.full"),
				HaveKeyWithValue("io.buildpacks.stack.description", "ubuntu:jammy with compilers and common libraries and utilities"),
				HaveKeyWithValue("io.buildpacks.stack.distro.name", "ubuntu"),
				HaveKeyWithValue("io.buildpacks.stack.distro.version", "22.04"),
				HaveKeyWithValue("io.buildpacks.stack.homepage", "https://github.com/paketo-buildpacks/jammy-full-stack"),
				HaveKeyWithValue("io.buildpacks.stack.maintainer", "Paketo Buildpacks"),
				HaveKeyWithValue("io.buildpacks.stack.metadata", MatchJSON("{}")),
			))

			buildReleaseDate, err = time.Parse(time.RFC3339, file.Config.Labels["io.buildpacks.stack.released"])
			Expect(err).NotTo(HaveOccurred())
			Expect(buildReleaseDate).NotTo(BeZero())

			Expect(image).To(SatisfyAll(
				HaveFileWithContent("/etc/group", ContainSubstring("cnb:x:1000:")),
				HaveFileWithContent("/etc/passwd", ContainSubstring("cnb:x:1001:1000::/home/cnb:/bin/bash")),
				HaveDirectory("/home/cnb"),
			))

			Expect(file.Config.User).To(Equal("1001:1000"))

			Expect(file.Config.Env).To(ContainElements(
				"CNB_USER_ID=1001",
				"CNB_GROUP_ID=1000",
				"CNB_STACK_ID=io.buildpacks.stacks.jammy.full",
			))

			Expect(image).To(HaveFileWithContent("/etc/gitconfig", ContainLines(
				"[safe]",
				"\tdirectory = /workspace",
				"\tdirectory = /workspace/source-ws",
				"\tdirectory = /workspace/source",
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status", SatisfyAll(
				ContainSubstring("Package: build-essential"),
				ContainSubstring("Package: ca-certificates"),
				ContainSubstring("Package: curl"),
				ContainSubstring("Package: git"),
				ContainSubstring("Package: jq"),
				ContainSubstring("Package: libgmp-dev"),
				ContainSubstring("Package: libssl3"),
				ContainSubstring("Package: libyaml-0-2"),
				ContainSubstring("Package: netbase"),
				ContainSubstring("Package: openssl"),
				ContainSubstring("Package: tzdata"),
				ContainSubstring("Package: xz-utils"),
				ContainSubstring("Package: zlib1g-dev"),
			)))
		})

		by("confirming that the run image is correct", func() {
			dir := filepath.Join(tmpDir, "run-index")
			err := os.Mkdir(dir, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			archive, err := os.Open(stack.RunArchive)
			Expect(err).NotTo(HaveOccurred())
			defer archive.Close()

			err = vacation.NewArchive(archive).Decompress(dir)
			Expect(err).NotTo(HaveOccurred())

			path, err := layout.FromPath(dir)
			Expect(err).NotTo(HaveOccurred())

			index, err := path.ImageIndex()
			Expect(err).NotTo(HaveOccurred())

			indexManifest, err := index.IndexManifest()
			Expect(err).NotTo(HaveOccurred())

			Expect(indexManifest.Manifests).To(HaveLen(1))
			Expect(indexManifest.Manifests[0].Platform).To(Equal(&v1.Platform{
				OS:           "linux",
				Architecture: "amd64",
			}))

			image, err := index.Image(indexManifest.Manifests[0].Digest)
			Expect(err).NotTo(HaveOccurred())

			file, err := image.ConfigFile()
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Config.Labels).To(SatisfyAll(
				HaveKeyWithValue("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy.full"),
				HaveKeyWithValue("io.buildpacks.stack.description", "ubuntu:jammy with common libraries and utilities"),
				HaveKeyWithValue("io.buildpacks.stack.distro.name", "ubuntu"),
				HaveKeyWithValue("io.buildpacks.stack.distro.version", "22.04"),
				HaveKeyWithValue("io.buildpacks.stack.homepage", "https://github.com/paketo-buildpacks/jammy-full-stack"),
				HaveKeyWithValue("io.buildpacks.stack.maintainer", "Paketo Buildpacks"),
				HaveKeyWithValue("io.buildpacks.stack.metadata", MatchJSON("{}")),
			))

			runReleaseDate, err = time.Parse(time.RFC3339, file.Config.Labels["io.buildpacks.stack.released"])
			Expect(err).NotTo(HaveOccurred())
			Expect(runReleaseDate).NotTo(BeZero())

			Expect(file.Config.User).To(Equal("1002:1000"))

			Expect(image).To(SatisfyAll(
				HaveFileWithContent("/etc/group", ContainSubstring("cnb:x:1000:")),
				HaveFileWithContent("/etc/passwd", ContainSubstring("cnb:x:1002:1000::/home/cnb:/sbin/nologin")),
				HaveDirectory("/home/cnb"),
			))

			Expect(image).To(SatisfyAll(
				HaveFile("/usr/share/doc/ca-certificates/copyright"),
				HaveFile("/etc/ssl/certs/ca-certificates.crt"),
				HaveDirectory("/root"),
				HaveDirectory("/tmp"),
				HaveFile("/etc/services"),
				HaveFile("/etc/nsswitch.conf"),
			))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/base-files", SatisfyAll(
				ContainSubstring("Package: base-files"),
				MatchRegexp("Version: [0-9]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: amd64"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/ca-certificates", SatisfyAll(
				ContainSubstring("Package: ca-certificates"),
				MatchRegexp("Version: [0-9]+"),
				ContainSubstring("Architecture: all"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/libc6", SatisfyAll(
				ContainSubstring("Package: libc6"),
				MatchRegexp("Version: [0-9\\.\\-]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: amd64"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/libssl3", SatisfyAll(
				ContainSubstring("Package: libssl3"),
				MatchRegexp("Version: [0-9\\.\\-]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: amd64"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/netbase", SatisfyAll(
				ContainSubstring("Package: netbase"),
				MatchRegexp("Version: [0-9\\.]+"),
				ContainSubstring("Architecture: all"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/openssl", SatisfyAll(
				ContainSubstring("Package: openssl"),
				MatchRegexp("Version: [0-9\\.\\-]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: amd64"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/tzdata", SatisfyAll(
				ContainSubstring("Package: tzdata"),
				MatchRegexp("Version: [a-z0-9\\.\\-]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: all"),
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status.d/zlib1g", SatisfyAll(
				ContainSubstring("Package: zlib1g"),
				MatchRegexp("Version: [a-z0-9\\.\\-\\:]+ubuntu[0-9\\.]+"),
				ContainSubstring("Architecture: amd64"),
			)))

			Expect(image).NotTo(HaveFile("/usr/share/ca-certificates"))

			Expect(image).To(HaveFileWithContent("/etc/os-release", SatisfyAll(
				ContainSubstring(`PRETTY_NAME="Paketo Buildpacks Full Jammy"`),
				ContainSubstring(`HOME_URL="https://github.com/paketo-buildpacks/jammy-full-stack"`),
				ContainSubstring(`SUPPORT_URL="https://github.com/paketo-buildpacks/jammy-full-stack/blob/main/README.md"`),
				ContainSubstring(`BUG_REPORT_URL="https://github.com/paketo-buildpacks/jammy-full-stack/issues/new"`),
			)))
		})
		Expect(runReleaseDate).To(Equal(buildReleaseDate))
	})
}
