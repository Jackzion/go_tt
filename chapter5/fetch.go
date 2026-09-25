package chapter5

import (
	"fmt"
	"io"
	"net/http"
	"os"
)

// fetch fetches the URL and returns the document
func Fetch(url string) string {

	// fetch html document
	resp, err := http.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		return ""
	}
	defer resp.Body.Close()
	// read html document
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fetch: %v\n", err)
		return ""
	}
	return string(body)
}
