package main

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"regexp"
	"strings"
	"text/template"
)

//go:embed template.md
var tString string

type Package struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
}

type ModifiedPackage struct {
	Name            string `json:"name"`
	PreviousVersion string `json:"previousVersion"`
	CurrentVersion  string `json:"currentVersion"`
	PreviousPURL    string `json:"previousPurl"`
	CurrentPURL     string `json:"currentPurl"`
}

type USN struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {

	var config struct {
		BuildImage                string
		RunImage                  string
		BuildPackagesAddedJSON    string
		BuildPackagesModifiedJSON string
		RunPackagesAddedJSON      string
		RunPackagesModifiedJSON   string
		PatchedJSON               string
	}

	flag.StringVar(&config.BuildImage, "build-image", "", "Registry location of stack build image")
	flag.StringVar(&config.RunImage, "run-image", "", "Registry location of stack run image")
	flag.StringVar(&config.PatchedJSON, "patched-usns", "", "JSON Array of patched USNs")
	flag.StringVar(&config.BuildPackagesAddedJSON, "build-added", "", "JSON Array of packages added to build image")
	flag.StringVar(&config.BuildPackagesModifiedJSON, "build-modified", "", "JSON Array of packages modified in build image")
	flag.StringVar(&config.RunPackagesAddedJSON, "run-added", "", "JSON Array of packages added to run image")
	flag.StringVar(&config.RunPackagesModifiedJSON, "run-modified", "", "JSON Array of packages modified in run image")
	flag.Parse()

	var contents struct {
		PatchedArray  []USN
		BuildAdded    []Package
		BuildModified []ModifiedPackage
		RunAdded      []Package
		RunModified   []ModifiedPackage
		BuildImage    string
		RunImage      string
	}

	err := json.Unmarshal([]byte(fixEmptyArray(config.PatchedJSON)), &contents.PatchedArray)
	if err != nil {
		log.Fatalf("failed unmarshalling patched USNs: %s", err.Error())
	}

	err = json.Unmarshal([]byte(fixEmptyArray(config.BuildPackagesAddedJSON)), &contents.BuildAdded)
	if err != nil {
		log.Fatalf("failed unmarshalling build packages added: %s", err.Error())
	}

	err = json.Unmarshal([]byte(fixEmptyArray(config.BuildPackagesModifiedJSON)), &contents.BuildModified)
	if err != nil {
		log.Fatalf("failed unmarshalling build packages modified: %s", err.Error())
	}

	err = json.Unmarshal([]byte(fixEmptyArray(config.RunPackagesAddedJSON)), &contents.RunAdded)
	if err != nil {
		log.Fatalf("failed unmarshalling run packages added: %s", err.Error())
	}

	err = json.Unmarshal([]byte(fixEmptyArray(config.RunPackagesModifiedJSON)), &contents.RunModified)
	if err != nil {
		log.Fatalf("failed unmarshalling run packages modified: %s", err.Error())
	}

	contents.BuildImage = config.BuildImage
	contents.RunImage = config.RunImage

	t, err := template.New("template.md").Parse(tString)
	if err != nil {
		log.Fatalf("failed to create release notes template: %s", err.Error())
	}

	b := bytes.NewBuffer(nil)
	err = t.Execute(b, contents)
	if err != nil {
		log.Fatalf("failed to execute release notes template: %s", err.Error())
	}

	fmt.Println(b.String())

	fmt.Println(fmt.Sprintf("::set-output name=release_body::%s", escape(b.String())))
}

func escape(original string) string {
	newline := regexp.MustCompile(`\n`)
	cReturn := regexp.MustCompile(`\r`)

	result := strings.ReplaceAll(original, `%`, `%25`)
	result = newline.ReplaceAllString(result, `%0A`)
	result = cReturn.ReplaceAllString(result, `%0D`)

	return result
}

func fixEmptyArray(original string) string {
	if original == "" {
		return `[]`
	}
	return original
}
