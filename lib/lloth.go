package lloth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"golang.org/x/net/html"
	"golang.org/x/sync/semaphore"

	_ "embed"
)

//go:embed cleaned_hosts.txt
var b []byte

type blocklist map[string]string

func newBlocklist() blocklist {
	hosts := strings.Split(string(b), "\n")
	bl := make(map[string]string)
	for _, host := range hosts {
		bl[host] = host
	}
	return bl
}

type LinkCollector struct {
	VisitedLinks map[string]struct{}
	DomainList   map[string]struct{}
	mu           sync.Mutex
	Wg           sync.WaitGroup // Exposing WaitGroup
	sem          *semaphore.Weighted
	blocklist
}

// NewLinkCollector initializes a new LinkCollector with the specified concurrency limit.
func NewLinkCollector(maxConcurrent int64) *LinkCollector {
	return &LinkCollector{
		VisitedLinks: make(map[string]struct{}),
		DomainList:   make(map[string]struct{}),
		sem:          semaphore.NewWeighted(maxConcurrent),
		blocklist:    newBlocklist(),
	}
}

// CollectLinks fetches the page and extracts links recursively.
func (lc *LinkCollector) CollectLinks(baseURL string) {
	u, _ := url.Parse(baseURL)
	if _, ok := lc.blocklist[u.Host]; ok {
		log.Println("Host", u.Host, "found in blocklist, skipping")
		return
	}
	defer lc.Wg.Done()

	// Acquire a semaphore to limit concurrent requests
	if err := lc.sem.Acquire(context.Background(), 1); err != nil {
		return
	}
	defer lc.sem.Release(1)

	// Fetch the page
	resp, err := http.Get(baseURL)
	if err != nil {
		fmt.Printf("CollectLinks: Error fetching %s: %s\n", baseURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("CollectLinks: Error received status code %d for %s\n", resp.StatusCode, baseURL)
		return
	}

	// Parse the HTML
	doc, err := html.Parse(resp.Body)
	if err != nil {
		fmt.Printf("CollectLinks: Error parsing HTML from %s: %s\n", baseURL, err)
		return
	}

	// Extract links from the document
	lc.ExtractLinks(doc, baseURL)

	lc.SaveVisitedLinks(u.Host + ".txt")
	lc.SaveDomainList("visited_domains.txt")
}

// ExtractLinks traverses the HTML nodes and adds links.
func (lc *LinkCollector) ExtractLinks(n *html.Node, baseURL string) {
	if n.Type == html.ElementNode {
		// Handle relevant tags for links
		switch n.Data {
		case "a":
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					lc.AddLink(attr.Val, baseURL)
				}
			}
		case "iframe":
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					lc.AddLink(attr.Val, baseURL)
				}
			}
		case "frame":
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					lc.AddLink(attr.Val, baseURL)
				}
			}
		case "script":
			for _, attr := range n.Attr {
				if attr.Key == "src" {
					lc.AddLink(attr.Val, baseURL)
				}
			}
		case "link":
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					lc.AddLink(attr.Val, baseURL)
				}
			}
		}
	}

	// Always traverse all children
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		lc.ExtractLinks(c, baseURL)
	}
}

// AddLink processes a link and initiates collection for new links.
func (lc *LinkCollector) AddLink(link, baseURL string) {
	parsedLink, err := url.Parse(link)
	if err == nil {
		if _, ok := lc.blocklist[parsedLink.Host]; ok {
			return
		}
		if parsedLink.Host == "" {
			// If it's a relative link, construct the full URL
			parsedBase, _ := url.Parse(baseURL)
			parsedLink = parsedBase.ResolveReference(parsedLink)
		}

		// Track the domain of the link
		lc.mu.Lock()
		if _, visited := lc.VisitedLinks[parsedLink.String()]; !visited {
			lc.VisitedLinks[parsedLink.String()] = struct{}{}
			lc.DomainList[parsedLink.Host] = struct{}{} // Track the domain
			lc.Wg.Add(1)                                // Increment the wait group counter
			go lc.CollectLinks(parsedLink.String())     // Recursively collect links
		}
		lc.mu.Unlock()
	}
}

// SaveVisitedLinks saves the visited links to a file.
func (lc *LinkCollector) SaveVisitedLinks(filename string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file %s: %s\n", filename, err)
		return
	}
	defer file.Close()

	for link := range lc.VisitedLinks {
		if _, err := file.WriteString(link + "\n"); err != nil {
			fmt.Printf("Error writing to file %s: %s\n", filename, err)
			return
		}
	}

	fmt.Printf("Visited links saved to %s\n", filename)
}

// SaveDomainList saves the unique domains visited to a JSON file.
func (lc *LinkCollector) SaveDomainList(filename string) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	domains := make([]string, 0, len(lc.DomainList))
	var badDomains []string
	for domain := range lc.DomainList {
		if _, ok := lc.blocklist[domain]; !ok {
			domains = append(domains, domain)
		}
		if ContainsAny(domain, anyof) {
			badDomains = append(badDomains, domain)
		}
	}

	file, err := os.Create(filename)
	if err != nil {
		fmt.Printf("Error creating file %s: %s\n", filename, err)
		return
	}
	defer file.Close()

	jsonData, err := json.MarshalIndent(domains, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON for domain list: %s\n", err)
		return
	}

	if _, err := file.Write(jsonData); err != nil {
		fmt.Printf("Error writing to file %s: %s\n", filename, err)
		return
	}
	if err := ioutil.WriteFile("bad_domains.txt", []byte(strings.Join(badDomains, "\n")), 0o644); err != nil {
		fmt.Printf("Error writing to file %s: %s\n", "bad_domains.txt", err)
		return
	}

	fmt.Printf("Visited domains saved to %s\n", filename)
}

var anyof = []string{"google.com", "msn.com", "github.com", "gitlab.com", "yahoo", "adtech", "ads.", "discord.gg", "discord.com", "yimg.com", "ytimg"}

func ContainsAny(str string, comp []string) bool {
	for _, c := range comp {
		if strings.Contains(str, c) && !strings.Contains(str, "downloads") {
			return true
		}
	}
	return false
}
