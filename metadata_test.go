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

			Expect(indexManifest.Manifests).To(HaveLen(2))

			platforms := []v1.Platform{}
			for _, manifest := range indexManifest.Manifests {
				platforms = append(platforms, v1.Platform{
					Architecture: manifest.Platform.Architecture,
					OS:           manifest.Platform.OS,
				})
			}
			Expect(platforms).To(ContainElement(v1.Platform{
				OS:           "linux",
				Architecture: "amd64",
			}))
			Expect(platforms).To(ContainElement(v1.Platform{
				OS:           "linux",
				Architecture: "arm64",
			}))

			image, err := index.Image(indexManifest.Manifests[0].Digest)
			Expect(err).NotTo(HaveOccurred())

			file, err := image.ConfigFile()
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Config.Labels).To(SatisfyAll(
				HaveKeyWithValue("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"),
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
				"CNB_STACK_ID=io.buildpacks.stacks.jammy",
			))

			Expect(image).To(HaveFileWithContent("/etc/gitconfig", ContainLines(
				"[safe]",
				"\tdirectory = /workspace",
				"\tdirectory = /workspace/source-ws",
				"\tdirectory = /workspace/source",
			)))

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status", SatisfyAll(
				ContainSubstring("Package: adduser"),
				ContainSubstring("Package: apt"),
				ContainSubstring("Package: base-files"),
				ContainSubstring("Package: base-passwd"),
				ContainSubstring("Package: bash"),
				ContainSubstring("Package: bsdutils"),
				ContainSubstring("Package: ca-certificates"),
				ContainSubstring("Package: coreutils"),
				ContainSubstring("Package: curl"),
				ContainSubstring("Package: dash"),
				ContainSubstring("Package: debconf"),
				ContainSubstring("Package: debianutils"),
				ContainSubstring("Package: diffutils"),
				ContainSubstring("Package: dpkg"),
				ContainSubstring("Package: e2fsprogs"),
				ContainSubstring("Package: findutils"),
				ContainSubstring("Package: gcc-12-base"),
				ContainSubstring("Package: git"),
				ContainSubstring("Package: git-man"),
				ContainSubstring("Package: gpgv"),
				ContainSubstring("Package: grep"),
				ContainSubstring("Package: gzip"),
				ContainSubstring("Package: hostname"),
				ContainSubstring("Package: init-system-helpers"),
				ContainSubstring("Package: libacl1"),
				ContainSubstring("Package: libapt-pkg6.0"),
				ContainSubstring("Package: libattr1"),
				ContainSubstring("Package: libaudit-common"),
				ContainSubstring("Package: libaudit1"),
				ContainSubstring("Package: libblkid1"),
				ContainSubstring("Package: libbrotli1"),
				ContainSubstring("Package: libbz2-1.0"),
				ContainSubstring("Package: libc-bin"),
				ContainSubstring("Package: libc6"),
				ContainSubstring("Package: libcap-ng0"),
				ContainSubstring("Package: libcap2"),
				ContainSubstring("Package: libcom-err2"),
				ContainSubstring("Package: libcrypt1"),
				ContainSubstring("Package: libcurl3-gnutls"),
				ContainSubstring("Package: libcurl4"),
				ContainSubstring("Package: libdb5.3"),
				ContainSubstring("Package: libdebconfclient0"),
				ContainSubstring("Package: liberror-perl"),
				ContainSubstring("Package: libexpat1"),
				ContainSubstring("Package: libext2fs2"),
				ContainSubstring("Package: libffi8"),
				ContainSubstring("Package: libgcc-s1"),
				ContainSubstring("Package: libgcrypt20"),
				ContainSubstring("Package: libgdbm-compat4"),
				ContainSubstring("Package: libgdbm6"),
				ContainSubstring("Package: libgmp10"),
				ContainSubstring("Package: libgnutls30"),
				ContainSubstring("Package: libgpg-error0"),
				ContainSubstring("Package: libgssapi-krb5-2"),
				ContainSubstring("Package: libhogweed6"),
				ContainSubstring("Package: libidn2-0"),
				ContainSubstring("Package: libk5crypto3"),
				ContainSubstring("Package: libkeyutils1"),
				ContainSubstring("Package: libkrb5-3"),
				ContainSubstring("Package: libkrb5support0"),
				ContainSubstring("Package: libldap-2.5-0"),
				ContainSubstring("Package: liblz4-1"),
				ContainSubstring("Package: liblzma5"),
				ContainSubstring("Package: libmount1"),
				ContainSubstring("Package: libncurses6"),
				ContainSubstring("Package: libncursesw6"),
				ContainSubstring("Package: libnettle8"),
				ContainSubstring("Package: libnghttp2-14"),
				ContainSubstring("Package: libnsl2"),
				ContainSubstring("Package: libp11-kit0"),
				ContainSubstring("Package: libpam-modules"),
				ContainSubstring("Package: libpam-modules-bin"),
				ContainSubstring("Package: libpam-runtime"),
				ContainSubstring("Package: libpam0g"),
				ContainSubstring("Package: libpcre2-8-0"),
				ContainSubstring("Package: libpcre3"),
				ContainSubstring("Package: libperl5.34"),
				ContainSubstring("Package: libprocps8"),
				ContainSubstring("Package: libpsl5"),
				ContainSubstring("Package: librtmp1"),
				ContainSubstring("Package: libsasl2-2"),
				ContainSubstring("Package: libsasl2-modules-db"),
				ContainSubstring("Package: libseccomp2"),
				ContainSubstring("Package: libselinux1"),
				ContainSubstring("Package: libsemanage-common"),
				ContainSubstring("Package: libsemanage2"),
				ContainSubstring("Package: libsepol2"),
				ContainSubstring("Package: libsmartcols1"),
				ContainSubstring("Package: libss2"),
				ContainSubstring("Package: libssh-4"),
				ContainSubstring("Package: libssl3"),
				ContainSubstring("Package: libstdc++6"),
				ContainSubstring("Package: libsystemd0"),
				ContainSubstring("Package: libtasn1-6"),
				ContainSubstring("Package: libtinfo6"),
				ContainSubstring("Package: libtirpc-common"),
				ContainSubstring("Package: libtirpc3"),
				ContainSubstring("Package: libudev1"),
				ContainSubstring("Package: libunistring2"),
				ContainSubstring("Package: libuuid1"),
				ContainSubstring("Package: libxxhash0"),
				ContainSubstring("Package: libzstd1"),
				ContainSubstring("Package: locales"),
				ContainSubstring("Package: login"),
				ContainSubstring("Package: logsave"),
				ContainSubstring("Package: lsb-base"),
				ContainSubstring("Package: mawk"),
				ContainSubstring("Package: mount"),
				ContainSubstring("Package: ncurses-base"),
				ContainSubstring("Package: ncurses-bin"),
				ContainSubstring("Package: openssl"),
				ContainSubstring("Package: passwd"),
				ContainSubstring("Package: perl"),
				ContainSubstring("Package: perl-base"),
				ContainSubstring("Package: perl-modules-5.34"),
				ContainSubstring("Package: procps"),
				ContainSubstring("Package: sed"),
				ContainSubstring("Package: sensible-utils"),
				ContainSubstring("Package: sysvinit-utils"),
				ContainSubstring("Package: tar"),
				ContainSubstring("Package: ubuntu-keyring"),
				ContainSubstring("Package: usrmerge"),
				ContainSubstring("Package: util-linux"),
				ContainSubstring("Package: zlib1g"),
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

			Expect(indexManifest.Manifests).To(HaveLen(2))

			platforms := []v1.Platform{}
			for _, manifest := range indexManifest.Manifests {
				platforms = append(platforms, v1.Platform{
					Architecture: manifest.Platform.Architecture,
					OS:           manifest.Platform.OS,
				})
			}
			Expect(platforms).To(ContainElement(v1.Platform{
				OS:           "linux",
				Architecture: "amd64",
			}))
			Expect(platforms).To(ContainElement(v1.Platform{
				OS:           "linux",
				Architecture: "arm64",
			}))

			image, err := index.Image(indexManifest.Manifests[0].Digest)
			Expect(err).NotTo(HaveOccurred())

			file, err := image.ConfigFile()
			Expect(err).NotTo(HaveOccurred())

			Expect(file.Config.Labels).To(SatisfyAll(
				HaveKeyWithValue("io.buildpacks.stack.id", "io.buildpacks.stacks.jammy"),
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
				HaveFileWithContent("/etc/passwd", ContainSubstring("cnb:x:1002:1000::/home/cnb:/bin/bash")),
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

			Expect(image).To(HaveFileWithContent("/var/lib/dpkg/status", SatisfyAll(
				ContainSubstring("Package: adduser"),
				ContainSubstring("Package: apt"),
				ContainSubstring("Package: base-files"),
				ContainSubstring("Package: base-passwd"),
				ContainSubstring("Package: bash"),
				ContainSubstring("Package: bsdutils"),
				ContainSubstring("Package: ca-certificates"),
				ContainSubstring("Package: coreutils"),
				ContainSubstring("Package: dash"),
				ContainSubstring("Package: debconf"),
				ContainSubstring("Package: debianutils"),
				ContainSubstring("Package: diffutils"),
				ContainSubstring("Package: dpkg"),
				ContainSubstring("Package: e2fsprogs"),
				ContainSubstring("Package: findutils"),
				ContainSubstring("Package: gcc-12-base"),
				ContainSubstring("Package: gpgv"),
				ContainSubstring("Package: grep"),
				ContainSubstring("Package: gzip"),
				ContainSubstring("Package: hostname"),
				ContainSubstring("Package: init-system-helpers"),
				ContainSubstring("Package: libacl1"),
				ContainSubstring("Package: libapt-pkg6.0"),
				ContainSubstring("Package: libattr1"),
				ContainSubstring("Package: libaudit-common"),
				ContainSubstring("Package: libaudit1"),
				ContainSubstring("Package: libblkid1"),
				ContainSubstring("Package: libbz2-1.0"),
				ContainSubstring("Package: libc-bin"),
				ContainSubstring("Package: libc6"),
				ContainSubstring("Package: libcap-ng0"),
				ContainSubstring("Package: libcap2"),
				ContainSubstring("Package: libcom-err2"),
				ContainSubstring("Package: libcrypt1"),
				ContainSubstring("Package: libdb5.3"),
				ContainSubstring("Package: libdebconfclient0"),
				ContainSubstring("Package: libext2fs2"),
				ContainSubstring("Package: libffi8"),
				ContainSubstring("Package: libgcc-s1"),
				ContainSubstring("Package: libgcrypt20"),
				ContainSubstring("Package: libgmp10"),
				ContainSubstring("Package: libgnutls30"),
				ContainSubstring("Package: libgpg-error0"),
				ContainSubstring("Package: libgssapi-krb5-2"),
				ContainSubstring("Package: libhogweed6"),
				ContainSubstring("Package: libidn2-0"),
				ContainSubstring("Package: libk5crypto3"),
				ContainSubstring("Package: libkeyutils1"),
				ContainSubstring("Package: libkrb5-3"),
				ContainSubstring("Package: libkrb5support0"),
				ContainSubstring("Package: liblz4-1"),
				ContainSubstring("Package: liblzma5"),
				ContainSubstring("Package: libmount1"),
				ContainSubstring("Package: libncurses6"),
				ContainSubstring("Package: libncursesw6"),
				ContainSubstring("Package: libnettle8"),
				ContainSubstring("Package: libnsl2"),
				ContainSubstring("Package: libp11-kit0"),
				ContainSubstring("Package: libpam-modules"),
				ContainSubstring("Package: libpam-modules-bin"),
				ContainSubstring("Package: libpam-runtime"),
				ContainSubstring("Package: libpam0g"),
				ContainSubstring("Package: libpcre2-8-0"),
				ContainSubstring("Package: libpcre3"),
				ContainSubstring("Package: libprocps8"),
				ContainSubstring("Package: libseccomp2"),
				ContainSubstring("Package: libselinux1"),
				ContainSubstring("Package: libsemanage-common"),
				ContainSubstring("Package: libsemanage2"),
				ContainSubstring("Package: libsepol2"),
				ContainSubstring("Package: libsmartcols1"),
				ContainSubstring("Package: libss2"),
				ContainSubstring("Package: libssl3"),
				ContainSubstring("Package: libstdc++6"),
				ContainSubstring("Package: libsystemd0"),
				ContainSubstring("Package: libtasn1-6"),
				ContainSubstring("Package: libtinfo6"),
				ContainSubstring("Package: libtirpc-common"),
				ContainSubstring("Package: libtirpc3"),
				ContainSubstring("Package: libudev1"),
				ContainSubstring("Package: libunistring2"),
				ContainSubstring("Package: libuuid1"),
				ContainSubstring("Package: libxxhash0"),
				ContainSubstring("Package: libzstd1"),
				ContainSubstring("Package: locales"),
				ContainSubstring("Package: login"),
				ContainSubstring("Package: logsave"),
				ContainSubstring("Package: lsb-base"),
				ContainSubstring("Package: mawk"),
				ContainSubstring("Package: mount"),
				ContainSubstring("Package: ncurses-base"),
				ContainSubstring("Package: ncurses-bin"),
				ContainSubstring("Package: openssl"),
				ContainSubstring("Package: passwd"),
				ContainSubstring("Package: perl-base"),
				ContainSubstring("Package: procps"),
				ContainSubstring("Package: sed"),
				ContainSubstring("Package: sensible-utils"),
				ContainSubstring("Package: sysvinit-utils"),
				ContainSubstring("Package: tar"),
				ContainSubstring("Package: ubuntu-keyring"),
				ContainSubstring("Package: usrmerge"),
				ContainSubstring("Package: util-linux"),
				ContainSubstring("Package: zlib1g"),
			)))

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
