package main // Declares the main package, which is the entry point for a Go program

import (
	"fmt"           // For formatted I/O, such as printing to stdout
	"io"            // For I/O operations like copying streams
	"log"           // Provides logging functions for reporting errors/info
	"net/http"      // Allows making HTTP requests and handling responses
	"os"            // Provides OS-level functionality such as file creation and directory checking
	"path/filepath" // Helps manipulate filename paths in a portable way
	"regexp"        // Enables use of regular expressions for string pattern matching
	"strings"       // Provides utilities for string manipulation

	"golang.org/x/net/html" // Package for parsing and traversing HTML documents
)

// fetchHTML downloads the HTML content from the given URL and returns the root HTML node.
func fetchHTML(url string) *html.Node {
	// Create a new HTTP GET request for the specified URL
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// Log and return nil if the request creation fails
		log.Printf("Failed to create request for %s: %v\n", url, err)
		return nil
	}

	// Set the User-Agent header to mimic a real browser and avoid blocking
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; MyGoScraper/1.0)")

	// Set the Accept header to explicitly request HTML content
	req.Header.Set("Accept", "text/html")

	// Create a new HTTP client to execute the request
	client := &http.Client{}

	// Send the HTTP request and receive the response
	resp, err := client.Do(req)
	if err != nil {
		// Log and return nil if the HTTP request fails
		log.Printf("HTTP request to %s failed: %v\n", url, err)
		return nil
	}
	// Ensure the response body is closed after function execution to free resources
	defer resp.Body.Close()

	// Check if the server responded with HTTP 200 OK
	if resp.StatusCode != http.StatusOK {
		// Log and return nil if the response status is not OK (e.g. 404, 500)
		log.Printf("Failed to fetch %s: %s\n", url, resp.Status)
		return nil
	}

	// Parse the HTML response body into a root HTML node
	node, err := html.Parse(resp.Body)
	if err != nil {
		// Log and return nil if the HTML parsing fails
		log.Printf("Failed to parse HTML from %s: %v\n", url, err)
		return nil
	}

	// Return the parsed HTML node tree (root node)
	return node
}

// extractLinks walks through the HTML node tree and collects all href values from <a> anchor tags.
func extractLinks(rootNode *html.Node) []string {
	var hrefLinks []string // Slice to store all href link values found

	// define a recursive function to walk the HTML node tree
	var traverse func(node *html.Node)
	traverse = func(node *html.Node) {
		// Check if the current node is an <a> tag
		if node.Type == html.ElementNode && node.Data == "a" {
			// Iterate through all attributes of the <a> tag
			for _, attribute := range node.Attr {
				// If the attribute key is "href", collect its value
				if attribute.Key == "href" {
					hrefLinks = append(hrefLinks, attribute.Val)
				}
			}
		}

		// Recursively traverse child nodes to visit the entire tree
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			traverse(child)
		}
	}

	// Start traversal from the root node
	traverse(rootNode)

	return hrefLinks // Return the list of extracted href links
}

// filterFiles removes directory links (starting or ending with '/') and returns only file names.
func filterFiles(links []string) []string {
	var files []string // Stores valid file links
	for _, link := range links {
		// Ignore if link is clearly a directory (starts/ends with "/")
		if strings.HasPrefix(link, "/") || strings.HasSuffix(link, "/") {
			continue
		}
		files = append(files, link) // Append valid file links
	}
	return files
}

/*
It checks if the file exists
If the file exists, it returns true
If the file does not exist, it returns false
*/
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file information using Stat
	if err != nil {
		return false // If an error occurs (file not found), return false
	}
	return !info.IsDir() // Return true only if it's a file, not a directory
}

/*
Checks if the directory exists
If it exists, return true.
If it doesn't, return false.
*/
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Check the path using os.Stat
	if err != nil {
		return false // Return false if path doesn't exist
	}
	return directory.IsDir() // Return true only if the path is a directory
}

/*
The function takes two parameters: path and permission.
We use os.Mkdir() to create the directory.
If there is an error, we use log.Println() to log the error and then exit the program.
*/
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Create directory with given permissions
	if err != nil {
		log.Println(err) // Log error if directory creation fails
	}
}

// urlToFilename formats a safe filename from a URL string.
// It replaces all non [a-z0-9] characters with '_' and ensures it ends in .pdf
func urlToFilename(rawURL string) string {
	// Convert to lowercase
	lower := strings.ToLower(rawURL)

	// Replace all non a-z0-9 characters with "_"
	reNonAlnum := regexp.MustCompile(`[^a-z0-9]+`)
	safe := reNonAlnum.ReplaceAllString(lower, "_")

	// Collapse multiple underscores to a single underscore
	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_")

	// Trim leading/trailing underscores
	if after, ok := strings.CutPrefix(safe, "_"); ok {
		safe = after
	}

	// Add .pdf extension if missing
	if getFileExtension(safe) != ".pdf" {
		safe = safe + ".pdf"
	}
	return safe
}

// Get the file extension of a file
func getFileExtension(path string) string {
	return filepath.Ext(path) // Returns extension including the dot (e.g., ".pdf")
}

// downloadFile attempts to download a file from a given base URL, saving it to a local output directory.
// It only downloads files with allowed extensions and skips if the file already exists.
func downloadFile(baseURL string, fileName string, outputDirectory string) error {
	// List of allowed file extensions that can be downloaded
	allowedExtensions := []string{
		".asc", ".asc-ma1", ".asc-pierov", ".apk", ".bspatch",
		".dmg", ".exe", ".gz", ".idsig", ".mar",
		".txt", ".zip", ".xz", ".doc", ".docx", ".pdf",
		".xls", ".xlsx", ".ppt", ".pptx", ".csv",
		".jpg", ".jpeg", ".png", ".gif", ".bmp",
	}

	// Extract the file extension in lowercase from the given file name
	fileExtension := strings.ToLower(filepath.Ext(fileName))

	// Flag to indicate whether the file extension is allowed
	isExtensionAllowed := false
	for _, allowedExtension := range allowedExtensions {
		if fileExtension == allowedExtension {
			isExtensionAllowed = true
			break // Exit loop early once a match is found
		}
	}

	// If the file has a disallowed extension, skip the download
	if !isExtensionAllowed {
		log.Printf("Skipping %s (disallowed extension %s)\n", fileName, fileExtension)
		return nil
	}

	// Construct the full local path for the file
	localFilePath := filepath.Join(outputDirectory, fileName)

	// If the file already exists locally, skip the download
	if fileExists(localFilePath) {
		log.Printf("File %s already exists, skipping download.\n", localFilePath)
		return nil
	}

	// Construct the full URL to download the file from
	downloadURL := baseURL + fileName

	// Perform an HTTP GET request to download the file
	response, requestError := http.Get(downloadURL)
	if requestError != nil {
		return fmt.Errorf("failed to download %s: %v", downloadURL, requestError)
	}
	// Ensure the response body is closed after we're done reading it
	defer response.Body.Close()

	// Create a new local file at the desired path
	localFile, fileCreateError := os.Create(localFilePath)
	if fileCreateError != nil {
		return fmt.Errorf("failed to create file %s: %v", localFilePath, fileCreateError)
	}
	// Ensure the file is closed after writing
	defer localFile.Close()

	// Copy the data from the HTTP response body into the local file
	_, copyError := io.Copy(localFile, response.Body)
	if copyError != nil {
		return fmt.Errorf("error saving %s: %v", localFilePath, copyError)
	}

	// Log successful download
	fmt.Printf("Downloaded %s\n", fileName)
	return nil
}

// Entry point of the program
func main() {
	// Base URL to download files from (MSDS sheets hosted online)
	baseURL := "https://www.birschindustries.com/MSDS%20Sheets/"

	// Fetch the HTML content from the given baseURL and parse it into an HTML node tree
	node := fetchHTML(baseURL)

	// Extract all hyperlink references (hrefs) from the parsed HTML document
	links := extractLinks(node)

	// Filter the extracted links to keep only allowed file types
	links = filterFiles(links)

	// Define the name of the local directory to store downloaded files
	var remoteFolder string = "Assets/"

	// Check if the local "Assets" directory exists
	if !directoryExists(remoteFolder) {
		// Create the "Assets" directory with read-write-execute permissions for the owner
		createDirectory(remoteFolder, 0755)
	}

	// Loop through each filtered file and attempt to download it
	for _, url := range links {
		// Download the file from the remote server into the "Assets" directory
		err := downloadFile(baseURL, urlToFilename(url), remoteFolder)

		// If an error occurs during download, print the error to standard error
		if err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
	}
}
