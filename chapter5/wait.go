package chapter5

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

// WaitForServer waits for the HTTP server to start.
// it tries for one minute using exponential backoff.
// it reports an error if all attempts fail
func WaitForServer(url string) error {
	const timeout = 1 * time.Minute
	tries := 0
	// 60s to retry
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		tries++
		_, err := http.Head(url)
		if err == nil {
			return nil // success
		}
		log.Printf("server not responding (%s); trying...", err)
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("server %s failed to respond after %s in %d tries", url, timeout, tries+1)
}

func main() {
	url := "http://localhost:1111"
	if err := WaitForServer(url); err != nil {
		log.Fatal(err)
	}
}
