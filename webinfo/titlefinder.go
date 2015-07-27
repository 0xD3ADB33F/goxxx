// The MIT License (MIT)
//
// Copyright (c) 2015 Romain LÉTENDART
//
// See LICENSE file.
package webinfo

import (
	"database/sql"
	"fmt"
	"github.com/thoj/go-ircevent"
	"golang.org/x/net/html"
	"golang.org/x/net/idna"
	"log"
	"net/http"
	"net/url"
	"regexp"
)

var _database *sql.DB

func Init(db *sql.DB) {
	_database = db
	sqlStmt := `CREATE TABLE IF NOT EXISTS Link (
    id integer NOT NULL PRIMARY KEY,
    user TEXT,
    url TEXT,
    date DATETIME DEFAULT CURRENT_TIMESTAMP);`

	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Fatalf("%q: %s\n", err, sqlStmt)
	}
}

// TODO Choose a better name for that function
func HandleUrls(event *irc.Event, replyCallback func(string)) {
	allUrls := findUrls(event.Message())
	for _, url := range allUrls {
		fmt.Println("Detected URL:", url.String())
		response, err := http.Get(url.String())
		if err != nil {
			log.Println(err)
			return
		}
		doc, err := html.Parse(response.Body)
		response.Body.Close()
		if err != nil {
			log.Println(err)
			return
		}

		var (
			user string
			date string
		)
		sqlQuery := "SELECT user, strftime('%d/%m/%Y @ %H:%M', datetime(date, 'localtime')) FROM Link WHERE url = $1"
		rows, err := _database.Query(sqlQuery, url.String())
		if err != nil {
			log.Fatalf("%q: %s\n", err, sqlQuery)
		}
		for rows.Next() {
			rows.Scan(&user, &date)
		}

		if user == "" {
			sqlQuery = "INSERT INTO Link (user, url) VALUES ($1, $2)"
			_, err := _database.Exec(sqlQuery, event.Nick, url.String())
			if err != nil {
				log.Fatalf("%q: %s\n", err, sqlQuery)
			}
		} else {
			replyCallback(fmt.Sprintf("Link already posted by %s (%s)", user, date))
		}

		title, found := getTitleFromHTML(doc)
		if found {
			replyCallback(title)
		}
	}
}

func getTitleFromHTML(document *html.Node) (title string, found bool) {
	if document.Type != html.DocumentNode {
		// Didn't find a document node as first node, exit
		return
	}

	// Try to find the <html> inside the document
	child := document.FirstChild
	for child != nil && !(child.Type == html.ElementNode && child.Data == "html") {
		child = child.NextSibling
	}
	if child == nil {
		// Didn't find <html>, exit
		return
	}

	// Try to find the <head> inside the document
	currentNode := child
	for child = currentNode.FirstChild; child != nil && !(child.Type == html.ElementNode && child.Data == "head"); child = child.NextSibling {
	}
	if child == nil {
		// Didn't find <head>, exit
		return
	}

	// Try to find the <title> inside the <head>
	currentNode = child
	for child = currentNode.FirstChild; child != nil && !(child.Type == html.ElementNode && child.Data == "title"); child = child.NextSibling {
	}
	if child == nil || child.FirstChild == nil {
		// Didn't find <title> or it is empty, exit
		return
	}

	// Retrieve the content inside the <title>
	title = child.FirstChild.Data
	found = true

	return
}

func findUrls(message string) (urls []*url.URL) {
	const maxUrlsCount int = 10

	// Source of the regular expression:
	// http://daringfireball.net/2010/07/improved_regex_for_matching_urls
	re := regexp.MustCompile("(?:https?://|www\\d{0,3}[.]|[a-z0-9.\\-]+[.][a-z]{2,4}/)(?:[^\\s()<>]+|\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\))+(?:\\(([^\\s()<>]+|(\\([^\\s()<>]+\\)))*\\)|[^\\s`!()\\[\\]{};:'\".,<>?«»“”‘’])")
	urlCandidates := re.FindAllString(message, maxUrlsCount)

	for _, candidate := range urlCandidates {
		url, err := url.Parse(candidate)
		if err != nil {
			break
		}
		// Scheme is required to query a URL
		if url.Scheme == "" {
			url.Scheme = "https"
		}
		// Conversion to ASCII is needed for Unicode hostnames
		asciiHost, err := idna.ToASCII(url.Host)
		if err == nil {
			url.Host = asciiHost
		}
		urls = append(urls, url)
	}

	return
}