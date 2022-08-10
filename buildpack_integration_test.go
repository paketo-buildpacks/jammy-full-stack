package acceptance_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/uuid"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"

	"github.com/paketo-buildpacks/occam"
	. "github.com/paketo-buildpacks/occam/matchers"
	"github.com/paketo-buildpacks/packit/v2/pexec"
)

func testBuildpackIntegration(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually

		buildPlanBuildpack string
		goDistBuildpack    string

		builderConfigFilepath string

		pack    occam.Pack
		docker  occam.Docker
		source  string
		name    string
		builder string

		image     occam.Image
		container occam.Container
	)

	it.Before(func() {
		pack = occam.NewPack().WithVerbose()
		docker = occam.NewDocker()

		var err error
		name, err = occam.RandomName()
		Expect(err).NotTo(HaveOccurred())

		buildpackStore := occam.NewBuildpackStore()

		buildPlanBuildpack, err = buildpackStore.Get.
			Execute("github.com/paketo-community/build-plan")
		Expect(err).NotTo(HaveOccurred())

		goDistBuildpack, err = buildpackStore.Get.
			WithVersion("1.2.3").
			Execute("github.com/paketo-buildpacks/go-dist")
		Expect(err).NotTo(HaveOccurred())

		source, err = occam.Source(filepath.Join("integration", "testdata", "simple_app"))
		Expect(err).NotTo(HaveOccurred())

		builderConfigFile, err := os.CreateTemp("", "builder.toml")
		Expect(err).NotTo(HaveOccurred())
		builderConfigFilepath = builderConfigFile.Name()

		_, err = fmt.Fprintf(builderConfigFile, `
[stack]
  build-image = "%s:latest"
  id = "io.buildpacks.stacks.jammy"
  run-image = "%s:latest"
`,
			stack.BuildImageID,
			stack.RunImageID,
		)
		Expect(err).NotTo(HaveOccurred())

		Expect(archiveToDaemon(stack.BuildArchive, stack.BuildImageID)).To(Succeed())
		Expect(archiveToDaemon(stack.RunArchive, stack.RunImageID)).To(Succeed())

		builder = fmt.Sprintf("builder-%s", uuid.NewString())
		logs, err := createBuilder(builderConfigFilepath, builder)
		Expect(err).NotTo(HaveOccurred(), logs)
	})

	it.After(func() {
		Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
		Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())

		lifecycleVersion, err := getLifecycleVersion(builder)
		Expect(err).NotTo(HaveOccurred())

		Expect(docker.Image.Remove.Execute(builder)).To(Succeed())
		Expect(os.RemoveAll(builderConfigFilepath)).To(Succeed())

		Expect(docker.Image.Remove.Execute(stack.BuildImageID)).To(Succeed())
		Expect(docker.Image.Remove.Execute(stack.RunImageID)).To(Succeed())

		Expect(docker.Image.Remove.Execute(fmt.Sprintf("buildpacksio/lifecycle:%s", lifecycleVersion))).To(Succeed())

		Expect(os.RemoveAll(source)).To(Succeed())
	})

	it("builds an app with a buildpack", func() {

		var err error
		var logs fmt.Stringer
		image, logs, err = pack.WithNoColor().Build.
			WithBuildpacks(
				goDistBuildpack,
				buildPlanBuildpack,
			).
			WithEnv(map[string]string{
				"BP_LOG_LEVEL": "DEBUG",
			}).
			WithPullPolicy("if-not-present").
			WithBuilder(builder).
			Execute(name, source)
		Expect(err).ToNot(HaveOccurred(), logs.String)

		container, err = docker.Container.Run.
			WithDirect().
			WithCommand("go").
			WithCommandArgs([]string{"run", "main.go"}).
			WithEnv(map[string]string{"PORT": "8080"}).
			WithPublish("8080").
			WithPublishAll().
			Execute(image.ID)
		Expect(err).NotTo(HaveOccurred())

		Eventually(container).Should(BeAvailable())
		Eventually(container).Should(Serve(MatchRegexp(`go1.*`)).OnPort(8080))
	})
}

func archiveToDaemon(path, id string) error {
	skopeo := pexec.NewExecutable("skopeo")

	return skopeo.Execute(pexec.Execution{
		Args: []string{
			"copy",
			fmt.Sprintf("oci-archive://%s", path),
			fmt.Sprintf("docker-daemon:%s:latest", id),
		},
	})
}

func createBuilder(config string, name string) (string, error) {
	buf := bytes.NewBuffer(nil)

	pack := pexec.NewExecutable("pack")
	err := pack.Execute(pexec.Execution{
		Stdout: buf,
		Stderr: buf,
		Args: []string{
			"builder",
			"create",
			name,
			fmt.Sprintf("--config=%s", config),
		},
	})
	return buf.String(), err
}

type Builder struct {
	LocalInfo struct {
		Lifecycle struct {
			Version string `json:"version"`
		} `json:"lifecycle"`
	} `json:"local_info"`
}

func getLifecycleVersion(builderID string) (string, error) {
	buf := bytes.NewBuffer(nil)
	pack := pexec.NewExecutable("pack")
	err := pack.Execute(pexec.Execution{
		Stdout: buf,
		Stderr: buf,
		Args: []string{
			"builder",
			"inspect",
			builderID,
			"-o",
			"json",
		},
	})

	if err != nil {
		return "", err
	}

	var builder Builder
	err = json.Unmarshal([]byte(buf.String()), &builder)
	if err != nil {
		return "", err
	}
	return builder.LocalInfo.Lifecycle.Version, nil
}
