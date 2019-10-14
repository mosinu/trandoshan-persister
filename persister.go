package main

import (
	"encoding/json"
	"fmt"
	"github.com/nats-io/nats.go"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	contentQueue   = "contentQueue"
	contentSubject = "contentSubject"
)

var (
	protocolRegex = regexp.MustCompile("https?://")
)

// Json data received from NATS
type resourceData struct {
	Url     string `json:"url"`
	Content string `json:"content"`
}

func main() {
	log.Print("Initializing persister")

	// connect to NATS server
	nc, err := nats.Connect(os.Getenv("NATS_URI"))
	if err != nil {
		log.Fatalf("Error while connecting to nats server: %s", err)
	}
	defer nc.Close()

	// initialize queue subscriber
	if _, err := nc.QueueSubscribe(contentSubject, contentQueue, handleMessages()); err != nil {
		log.Fatalf("Error while trying to subscribe to server: %s", err)
	}

	log.Print("Consumer initialized successfully")

	// todo: better way
	select {}
}

func handleMessages() func(*nats.Msg) {
	return func(msg *nats.Msg) {
		var data resourceData

		// Unmarshal message
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Printf("Error while de-serializing payload: %sf", err)
			// todo: store in sort of DLQ?
			return
		}

		// Store content in the filesystem
		currentTime := time.Now()
		filePath := fmt.Sprintf("%s/%s", os.Getenv("STORAGE_PATH"), computePath(data.Url, currentTime))
		log.Printf("Storing content on path: %s", filePath)

		if err := ioutil.WriteFile(filePath, []byte(data.Content), 0644); err != nil {
			log.Printf("Error while trying to save content: %s", err)
			return
		}

		// todo call Elasticsearch and create model from content
	}
}

// Compute path for resource storage using his URL and the crawling time
// Format is: resource-url/64bit-timestamp
// f.e: http://login.google.com/secure/createAccount.html -> login.google.com/secure/createAccount.html/1570788418
func computePath(resourceUrl string, crawlData time.Time) string {
	// first of all sanitize resource URL
	var sanitizedResourceUrl string
	// remove protocol
	sanitizedResourceUrl = protocolRegex.ReplaceAllLiteralString(resourceUrl, "")
	// remove any trailing '/'
	sanitizedResourceUrl = strings.TrimSuffix(sanitizedResourceUrl, "/")

	return fmt.Sprintf("%s/%d", sanitizedResourceUrl, crawlData.Unix())
}

// Extract title from given html
func extractTitle(body string) string {
	cleanBody := strings.ToLower(body)
	startPos := strings.Index(cleanBody, "<title>") + len("<title>")
	endPos := strings.Index(cleanBody, "</title>")

	// html tag absent of malformed
	if startPos == -1 || endPos == -1 {
		return ""
	}
	return body[startPos:endPos]
}
