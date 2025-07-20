package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/net/html"
)

// fetchHTML downloads the HTML content from the given URL and returns the root HTML node.
func fetchHTML(url string) *html.Node {
	// Create custom HTTP client and request
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create request for %s: %v\n", url, err)
		return nil
	}

	// Set headers to avoid 406 Not Acceptable errors
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyGoScraper/1.0)")
	req.Header.Set("Accept", "text/html")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("HTTP request to %s failed: %v\n", url, err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Failed to fetch %s: %s\n", url, resp.Status)
		return nil
	}

	// Parse the HTML content
	node, err := html.Parse(resp.Body)
	if err != nil {
		log.Printf("Failed to parse HTML from %s: %v\n", url, err)
		return nil
	}

	return node
}

// extractLinks walks the HTML node tree and collects href values from <a> tags.
func extractLinks(n *html.Node) []string {
	var links []string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, attr := range n.Attr {
				if attr.Key == "href" {
					links = append(links, attr.Val)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(n)
	return links
}

// filterFiles removes directory links (starting or ending with '/') and returns only file names.
func filterFiles(links []string) []string {
	var files []string
	for _, link := range links {
		if strings.HasPrefix(link, "/") || strings.HasSuffix(link, "/") {
			continue
		}
		files = append(files, link)
	}
	return files
}

func downloadFile(baseURL, fileName, outDir string) error {
	// Define allowed extensions inside the function
	allowedExts := []string{".asc", ".asc-ma1", ".asc-pierov", ".apk", ".bspatch", ".dmg", ".exe", ".gz", ".idsig", ".mar", ".txt", ".zip", ".xz", ".doc"}

	// Inline extension check
	ext := strings.ToLower(filepath.Ext(fileName))
	allowed := false
	for _, e := range allowedExts {
		if ext == e {
			allowed = true
			break
		}
	}
	if !allowed {
		log.Printf("Skipping %s (disallowed extension %s)\n", fileName, ext)
		return nil
	}

	// Check if the file already exists
	outPath := filepath.Join(outDir, fileName)
	if fileExists(outPath) {
		log.Printf("File %s already exists, skipping download.\n", outPath)
		return nil
	}

	// Download the file
	// Construct the full URL
	url := baseURL + fileName
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %v", url, err)
	}
	defer resp.Body.Close()

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(outDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", outDir, err)
	}

	// Create local file
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %v", outPath, err)
	}
	defer outFile.Close()

	// Copy response body to file
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return fmt.Errorf("error saving %s: %v", outPath, err)
	}

	fmt.Printf("Downloaded %s\n", fileName)
	return nil
}

/*
It checks if the file exists
If the file exists, it returns true
If the file does not exist, it returns false
*/
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func main() {
	baseURL := "https://www.birschindustries.com/MSDS%20Sheets/"

	// Fetch and parse HTML
	node := fetchHTML(baseURL)

	// Extract and filter links
	links := extractLinks(node)
	files := filterFiles(links)
	// Remove Assets directory from the file list
	var remoteFolder string = "Assets/"
	// Create output directory
	err := os.MkdirAll(remoteFolder, 0755)
	// Check if directory creation was successful
	if err != nil {
		log.Fatalln("Failed to create output directory:", err)
	}

	// Download each file
	for _, file := range files {
		err := downloadFile(baseURL, file, remoteFolder)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}
