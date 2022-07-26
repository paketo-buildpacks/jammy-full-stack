package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mmcdole/gofeed"
)

var distroToVersionRegex map[string]string = map[string]string{
	"jammy":  `22\.04`,
	"focal":  `20\.04`,
	"bionic": `18\.04`,
}

type USN struct {
	AffectedPackages []string `json:"affected_packages"`
	CVEs             []CVE    `json:"cves"`
	Title            string   `json:"title"`
	ID               string   `json:"id"`
	URL              url.URL  `json:"-"`
	URLString        string   `json:"url"`
}

type CVE struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

func main() {
	var config struct {
		Distro       string
		LastUSNsJSON string
		Output       string
		PackagesJSON string
		RSSURL       string
	}

	flag.StringVar(&config.LastUSNsJSON,
		"last-usns",
		"",
		"JSON array of last known USNs")
	flag.StringVar(&config.RSSURL,
		"feed-url",
		"https://ubuntu.com/security/notices/rss.xml",
		"URL of RSS feed")
	flag.StringVar(&config.PackagesJSON,
		"packages",
		"",
		"JSON array of relevant packages")
	flag.StringVar(&config.Distro,
		"distro",
		`bionic`,
		"Name of Ubuntu distro: jammy, bionic")
	flag.StringVar(&config.Output,
		"output",
		"",
		"Path to output JSON file")

	flag.Parse()

	if config.LastUSNsJSON == "" {
		config.LastUSNsJSON = `[]`
	}

	if config.PackagesJSON == "" {
		config.PackagesJSON = `[]`
	}

	var lastUSNs []USN
	err := json.Unmarshal([]byte(config.LastUSNsJSON), &lastUSNs)
	if err != nil {
		log.Fatal(err)
	}

	var packages []string
	err = json.Unmarshal([]byte(config.PackagesJSON), &packages)
	if err != nil {
		log.Fatal(err)
	}

	newUSNs, err := getNewUSNsFromFeed(config.RSSURL, lastUSNs, distroToVersionRegex[config.Distro])
	if err != nil {
		log.Fatal(err)
	}

	filtered := filterUSNsByPackages(newUSNs, packages)

	fmt.Println("Getting CVE metadata for relevant USNs...")
	for i := range filtered {
		err = addCVEs(&filtered[i])
		if err != nil {
			log.Fatal(err)
		}
	}

	output, err := json.Marshal(filtered)
	if err != nil {
		log.Fatal(err)
	}

	if len(filtered) == 0 {
		output = []byte(`[]`)
	}

	fmt.Println("Output: ", string(output))
	fmt.Printf("::set-output name=usns::%s\n", string(output))

	if config.Output != "" {
		path, err := filepath.Abs(config.Output)
		if err != nil {
			log.Fatal(err)
		}
		err = os.WriteFile(path, output, os.ModePerm)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func filterUSNsByPackages(usns []USN, packages []string) (filtered []USN) {
	if len(packages) == 0 {
		fmt.Println("No packages specified. Skipping filtering.")
		return usns
	}

	fmt.Println("Filtering USNs by affected packages...")
	for _, usn := range usns {
	matchPkgs:
		for _, affected := range usn.AffectedPackages {
			for _, pkg := range packages {
				if pkg == affected {
					filtered = append(filtered, usn)
					fmt.Printf("USN '%s' contains affected package '%s'\n", usn.Title, affected)
					break matchPkgs
				}
			}
		}
	}
	return filtered
}

func addCVEs(usn *USN) error {
	usnBody, code, err := get(usn.URL.String())
	if err != nil {
		return fmt.Errorf("error getting USN: %w", err)
	}

	if code != http.StatusOK {
		return fmt.Errorf("unexpected status code getting USN: %d", code)
	}

	cves, err := extractCVEs(usnBody, usn.URL)
	if err != nil {
		return fmt.Errorf("error extracting CVEs from USN %s: %w", usn.Title, err)
	}
	usn.CVEs = cves
	return nil
}

func getNewUSNsFromFeed(rssURL string, lastUSNs []USN, distro string) ([]USN, error) {
	fp := gofeed.NewParser()
	feed, err := fp.ParseURL(rssURL)
	if err != nil {
		return nil, fmt.Errorf("error parsing rss feed: %w", err)
	}

	fmt.Println("Looking for new USNs...")
	var feedUSNs []USN
	for _, item := range feed.Items {
		// regex extracts 'USN-5464-1' from 'USN-5464-1: e2fsprogs vulnerability'
		re := regexp.MustCompile(`USN\-\d+\-\d+`)
		id := re.FindString(item.Title)

		// Matching on IDs is stricter since titles are sometimes edited after
		// publication. Matching on titles guards against ID parsing errors.
		if (len(lastUSNs) > 0) && (id == lastUSNs[0].ID || item.Title == lastUSNs[0].Title) {
			fmt.Println("No more new USNs.")
			break
		}
		fmt.Printf("New USN found: %s\n", item.Title)

		usnURL, err := url.Parse(item.Link)
		if err != nil {
			return nil, fmt.Errorf("error parsing URL of USN %s: %w", item.Title, err)
		}

		usnBody, code, err := get(usnURL.String())
		if err != nil {
			return nil, fmt.Errorf("error getting USN: %w", err)
		}

		if code != http.StatusOK {
			return nil, fmt.Errorf("unexpected status code getting USN: %d", code)
		}

		feedUSNs = append(feedUSNs, USN{
			ID:               id,
			Title:            item.Title,
			URL:              *usnURL,
			URLString:        usnURL.String(),
			AffectedPackages: getAffectedPackages(usnBody, distro),
		})
	}

	return feedUSNs, nil
}

func getAffectedPackages(usnBody, versionRegex string) []string {
	re := regexp.MustCompile("Update instructions</h2>(.*?)References")
	packagesList := re.FindString(usnBody)

	re = regexp.MustCompile(fmt.Sprintf(`%s.*?</ul>`, versionRegex))
	bionicPackages := re.FindString(packagesList)

	re = regexp.MustCompile(`<li class="p-list__item">(.*?)</li>`)
	listMatches := re.FindAllStringSubmatch(bionicPackages, -1)

	packages := make([]string, 0)
	for _, listItem := range listMatches {
		packages = append(packages, getPackageNameFromHTML(strings.TrimSpace(listItem[1])))
	}

	return packages
}

func get(url string) (string, int, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", 0, err
	}

	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", 0, err
	}

	body := html.UnescapeString(string(respBody))
	body = strings.ReplaceAll(body, "\n", " ")
	body = strings.ReplaceAll(body, "<br />", " ")
	body = strings.ReplaceAll(body, "<br>", " ")
	body = strings.ReplaceAll(body, "</br>", " ")

	return body, resp.StatusCode, nil
}
func extractCVEs(usnBody string, usnURL url.URL) ([]CVE, error) {

	// regex matches '<a href="/security/CVE-2022-1664">CVE-2022-1664</a>' or
	// '<a href="/cve/CVE-2022-1664">CVE-2022-1664</a>'
	re := regexp.MustCompile(`<a.*?href="([\S]*?(:?cve|security)\/CVE.*?)">(.*?)<\/a.*?>`)
	cves := re.FindAllStringSubmatch(usnBody, -1)

	re = regexp.MustCompile(`.*?href="([\S]*?launchpad\.net/bugs.*?)">(.*?)</li`)
	lps := re.FindAllStringSubmatch(usnBody, -1)

	var cveArray []CVE
	for _, cve := range cves {
		cveURL := url.URL{
			Scheme: "https",
			Host:   usnURL.Hostname(),
			Path:   cve[1],
		}

		cveArray = append(cveArray, CVE{
			Title: cve[3],
			URL:   cveURL.String(),
		})
	}

	for _, lp := range lps {
		cveArray = append(cveArray, CVE{
			Title: lp[2],
			URL:   lp[1],
		})
	}

	return cveArray, nil
}

func getPackageNameFromHTML(listItem string) string {
	if strings.HasPrefix(listItem, "<a href=") {
		re := regexp.MustCompile(`<a href=".*?">(.*?)</a>`)
		packageMatch := re.FindStringSubmatch(listItem)
		return packageMatch[1]
	}
	return strings.Split(listItem, " ")[0]
}
