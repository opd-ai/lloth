package main

import (
	"flag"
	"fmt"
	"net/url"

	lloth "github.com/opd-ai/lloth/lib"
)

var uri = flag.String("url", "https://example.com", "start URL")

func main() {
	flag.Parse()
	startURL := *uri
	parsedURL, err := url.Parse(startURL)
	if err != nil {
		fmt.Printf("Error parsing start URL: %s\n", err)
		return
	}

	maxConcurrent := int64(5) // Set the max number of concurrent goroutines
	linkCollector := lloth.NewLinkCollector(maxConcurrent)
	linkCollector.Wg.Add(1) // Start the first goroutine
	go linkCollector.CollectLinks(parsedURL.String())

	// Wait for all goroutines to finish
	linkCollector.Wg.Wait()
	linkCollector.SaveDomainList("visited_domains.json") // Save the list of crawled domains
	fmt.Println("All links collected.")
}
