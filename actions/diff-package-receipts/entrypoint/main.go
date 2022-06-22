package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

type CycloneDXPackageList struct {
	Components []CycloneDXComponent `json:"components"`
}

type CycloneDXComponent struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	PURL    string `json:"purl"`
}

type ModifiedCycloneDXComponent struct {
	Name            string `json:"name"`
	PreviousVersion string `json:"previousVersion"`
	CurrentVersion  string `json:"currentVersion"`
	PreviousPURL    string `json:"previousPurl"`
	CurrentPURL     string `json:"currentPurl"`
}

func main() {
	var config struct {
		CurrentPath  string
		PreviousPath string
	}

	flag.StringVar(&config.PreviousPath,
		"previous",
		"",
		"Path to previous package receipt")

	flag.StringVar(&config.CurrentPath,
		"current",
		"",
		"Path to current package receipt")

	flag.Parse()

	if config.CurrentPath == "" || config.PreviousPath == "" {
		log.Fatal("Must provide current and previous paths")
	}

	absolute, err := filepath.Abs(config.CurrentPath)
	if err != nil {
		log.Fatalf("Failed to create absolute path for %s", config.CurrentPath)
	}
	config.CurrentPath = absolute

	absolute, err = filepath.Abs(config.PreviousPath)
	if err != nil {
		log.Fatalf("Failed to create absolute path for %s", config.PreviousPath)
	}
	config.PreviousPath = absolute

	previous, err := parsePackagesFromFile(config.PreviousPath)
	if err != nil {
		log.Fatal(err)
	}
	current, err := parsePackagesFromFile(config.CurrentPath)
	if err != nil {
		log.Fatal(err)
	}

	var added, removed []CycloneDXComponent
	var modified []ModifiedCycloneDXComponent
	for prevName, prevPackage := range previous {
		if _, ok := current[prevName]; !ok {
			// package in previous but not in current
			removed = append(removed, prevPackage)
			continue
		}
		// package appears in both previous and current
		curPackage := current[prevName]
		if prevPackage.Version != curPackage.Version || prevPackage.PURL != curPackage.PURL {
			// package metadata has changed
			modified = append(modified, ModifiedCycloneDXComponent{
				Name:            curPackage.Name,
				PreviousVersion: prevPackage.Version,
				CurrentVersion:  prevPackage.Version,
				PreviousPURL:    prevPackage.PURL,
				CurrentPURL:     curPackage.PURL,
			})
		}
	}

	for curName, curPackage := range current {
		if _, ok := previous[curName]; !ok {
			// package appears in current, not in previous
			added = append(added, curPackage)
		}
	}

	addedJSON, err := json.Marshal(added)
	if err != nil {
		log.Fatal(err)
	}
	if len(added) == 0 {
		addedJSON = []byte(`[]`)
	}

	removedJSON, err := json.Marshal(removed)
	if err != nil {
		log.Fatal(err)
	}
	if len(removed) == 0 {
		removedJSON = []byte(`[]`)
	}

	modifiedJSON, err := json.Marshal(modified)
	if err != nil {
		log.Fatal(err)
	}
	if len(modified) == 0 {
		modifiedJSON = []byte(`[]`)
	}

	fmt.Println("Added packages:")
	for _, pkg := range added {
		fmt.Println(pkg.Name, pkg.Version)
	}
	fmt.Println("Removed packages:")
	for _, pkg := range removed {
		fmt.Println(pkg.Name, pkg.Version)
	}
	fmt.Println("Modified packages:")
	for _, pkg := range modified {
		fmt.Printf("%[1]s %s (PURL: %s) => %[1]s %s (PURL: %s)\n",
			pkg.Name,
			pkg.PreviousVersion,
			pkg.PreviousPURL,
			pkg.CurrentVersion,
			pkg.CurrentPURL,
		)
	}

	fmt.Printf("::set-output name=added::%s\n", string(addedJSON))
	fmt.Printf("::set-output name=removed::%s\n", string(removedJSON))
	fmt.Printf("::set-output name=modified::%s\n", string(modifiedJSON))
}

func parsePackagesFromFile(path string) (map[string]CycloneDXComponent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to open %s: %w", path, err)
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var c CycloneDXPackageList
	err = dec.Decode(&c)
	if err != nil {
		return nil, err
	}

	packages := make(map[string]CycloneDXComponent)
	for _, component := range c.Components {
		packages[component.Name] = component
	}

	return packages, nil
}
