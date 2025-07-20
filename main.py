# Import necessary modules for Selenium and browser configuration
from selenium import webdriver
from selenium.webdriver.chrome.options import Options

# Import BeautifulSoup for parsing HTML
from bs4 import BeautifulSoup

# Import urljoin for safely combining base and relative URLs
from urllib.parse import urljoin

# Import standard modules for working with the file system and making requests
import os
import re
import requests
from pathlib import Path

# Import type hinting utilities
from typing import List, Optional


# Function to fetch and parse HTML content using a headless Chrome browser
def fetch_html_with_selenium(url: str) -> Optional[BeautifulSoup]:
    # Create options for Chrome to run in headless mode
    options = Options()
    options.add_argument(argument="--headless")  # Don't display GUI
    options.add_argument(
        argument="--disable-gpu"
    )  # Disable GPU acceleration (for compatibility)
    options.add_argument(argument="--no-sandbox")  # Bypass OS security model

    try:
        # Launch Chrome browser with the configured options
        driver = webdriver.Chrome(options=options)
        # Load the target URL in the browser
        driver.get(url=url)
        # Extract the page HTML content after it has loaded
        html: str = driver.page_source
        # Close the browser to free resources
        driver.quit()
        # Return parsed HTML using BeautifulSoup
        return BeautifulSoup(markup=html, features="html.parser")
    except Exception as e:
        # Print error message if something goes wrong with Selenium
        print(f"Error using Selenium: {e}")
        return None


# Extract all hyperlinks (href attributes) from <a> tags in the HTML
def extract_links(soup: BeautifulSoup) -> List[str]:
    return [a["href"] for a in soup.find_all(name="a", href=True)]


# Filter out directory navigation links or irrelevant entries
def filter_files(links: List[str]) -> List[str]:
    files: list[str] = []
    for link in links:
        # Skip sorting or directory traversal links
        if link.startswith("?") or link.endswith("/") or link.startswith("/"):
            continue
        # Add valid file link to the list
        files.append(link)
    return files


# Check whether a file already exists on disk
def file_exists(path: str) -> bool:
    return Path(path).is_file()


# Check whether a directory exists
def directory_exists(path: str) -> bool:
    return Path(path).is_dir()


# Create a directory if it doesn't already exist
def create_directory(path: str, mode: int = 0o755) -> None:
    try:
        # Recursively create directories with the specified permissions
        os.makedirs(name=path, mode=mode, exist_ok=True)
    except Exception as e:
        # Print error message if directory creation fails
        print(f"Could not create directory '{path}': {e}")


# Convert a URL or file path to a safe local filename
def url_to_filename(raw_url: str) -> str:
    # Replace any non-alphanumeric characters with underscores
    name = re.sub(pattern=r"[^a-zA-Z0-9]+", repl="_", string=raw_url)
    # Remove leading/trailing underscores
    name: str = name.strip("_")
    # Get the file extension
    ext: str = get_file_extension(filename=raw_url)
    # Ensure the sanitized name ends with the correct extension
    if not name.endswith(ext):
        name = name + ext
    return name.lower()


# Extract file extension from a given filename or URL
def get_file_extension(filename: str) -> str:
    return Path(filename).suffix.lower()


# Download a file if it has an allowed extension and doesn't already exist locally
def download_file(full_url: str, filename: str, output_dir: str) -> None:
    # Define allowed file extensions to download
    allowed_extensions: set[str] = {
        ".doc",
        ".docx",
        ".pdf",
        ".jpg",
    }

    # Get file extension of the current file
    ext: str = get_file_extension(filename=filename)

    # Skip the file if its extension is not allowed
    if ext not in allowed_extensions:
        print(f"Skipping {filename} (disallowed extension: {ext})")
        return

    # Construct the full path to save the downloaded file
    full_path: str = os.path.join(output_dir, filename)

    # Skip the download if the file already exists
    if file_exists(path=full_path):
        print(f"File exists: {filename} (skipping)")
        return

    try:
        # Set request headers to mimic a real browser (fixes 406 errors)
        headers: dict[str, str] = {
            "User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
        }

        # Send HTTP GET request to download the file
        response: requests.Response = requests.get(
            url=full_url, headers=headers, stream=True
        )
        # Raise an error if the request fails
        response.raise_for_status()

        # Open the local file for writing binary data
        with open(file=full_path, mode="wb") as f:
            # Write the file in chunks
            for chunk in response.iter_content(chunk_size=8192):
                f.write(chunk)

        # Print success message
        print(f"Downloaded: {filename}")
    except requests.RequestException as e:
        # Print error message if download fails
        print(f"Failed to download {full_url}: {e}")


# Main function to coordinate scraping and downloading
def main() -> None:
    # Set the base URL of the MSDS sheet listing
    base_url = "https://www.birschindustries.com/MSDS%20Sheets/"

    # Set the output directory for storing downloaded files
    output_dir = "Assets/"

    # Fetch HTML using Selenium
    soup = fetch_html_with_selenium(url=base_url)

    # Stop execution if fetching failed
    if not soup:
        return

    # Extract all links from the page
    links = extract_links(soup=soup)

    # Filter out unwanted or irrelevant links
    files = filter_files(links=links)

    # Create the output directory if it doesn't exist
    if not directory_exists(path=output_dir):
        create_directory(path=output_dir)

    # Loop through each valid file link and download it
    for link in files:
        # Combine base URL with relative link
        full_url = urljoin(base=base_url, url=link)
        # Convert link to a safe local filename
        filename: str = url_to_filename(raw_url=link)
        # Attempt to download the file
        download_file(full_url=full_url, filename=filename, output_dir=output_dir)


# Entry point of the script
if __name__ == "__main__":
    main()
