package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/mvdan/xurls"
	"github.com/syndtr/goleveldb/leveldb"

	"golang.org/x/net/idna"
)

// var (
// link = flag.String("link", "", "link to parsing")
// originText = flag.String("originText", "", "origin text ")
// )
var parserDB *leveldb.DB

func printHeaders(headers http.Header) {
	log.Println("Starting print headers")

	for key, val := range headers {
		log.Printf("%s : %s", key, val)
	}

	log.Println("Finished print headers")
}

// JSONMessage is info about url
type JSONMessage struct {
	Title         string
	Type          string
	URL           string
	ThumbnailURL  string
	ContentType   string
	ContentLength int
}

func validateURL(link string, baseURL *url.URL) (string, error) {
	if link == "" {
		return "", errors.New("can't validate empty link")
	}

	link = strings.TrimSpace(link)

	links := xurls.Relaxed.FindAllString(link, -1)

	var u *url.URL
	var err error
	if len(links) == 1 {
		//in this case i guess that it is url
		link = links[0]

		u, err = url.Parse(link)
		if err != nil {
			//something goes wrong
			log.Println(err)

			return "", err
		}

		if u.Scheme == "" {
			//in this case i guess that url doesn't have scheme
			link = "http://" + link

			if u, err = url.ParseRequestURI(link); err != nil {
				log.Println(err)

				return "", err
			}
		}

		if u.Host == "jira.vibelab.net" {
			return "", errors.New("host in black list")
		}
	} else if len(links) == 0 {
		//in this case i guess that it is relative path
		if baseURL == nil {
			return "", errors.New("can't resolve relative url, base url is nil")
		}

		u, err = url.Parse(link)
		if err != nil {
			//something goes wrong
			log.Println(err)

			return "", err
		}

		u = baseURL.ResolveReference(u)
	} else {
		return "", errors.New("it can't be validated")
	}

	//convert internationalized domain to ascii
	u.Host, err = idna.ToASCII(strings.ToLower(u.Host))
	if err != nil {
		log.Println(err)

		return "", err
	}

	return u.String(), nil
}

func networkRequest(link string, header map[string]string, redirectedLink *string) (*http.Response, error) {
	jar, err := cookiejar.New(nil)

	redirectHandler := func(req *http.Request, via []*http.Request) error {
		if redirectedLink != nil {
			*redirectedLink = req.URL.String()
		}

		return nil
	}

	client := &http.Client{Jar: jar, Timeout: time.Second * 30, CheckRedirect: redirectHandler}

	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		log.Println(err)

		return nil, err
	}

	req.Header.Add("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_10_3) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/43.0.2357.81 Safari/537.36")
	req.Header.Add("accept", "*/*")

	for key, val := range header {
		req.Header.Add(key, val)
	}

	req.Close = true

	resp, err := client.Do(req)

	if err != nil {
		log.Println(err)

		return nil, err
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusPartialContent {
		errStr := fmt.Sprintf("Status for link %s is %s", link, resp.Status)

		log.Println(errStr)

		defer resp.Body.Close()

		return nil, errors.New(errStr)
	}

	return resp, err
}

func processLink(link string) ([]byte, error) {
	direct := make(chan *JSONMessage)
	go func() {
		direct <- directLinkProcessing(link)

		defer close(direct)
	}()

	common := make(chan *JSONMessage)
	go func() {
		common <- commonLinkProcessing(link)

		defer close(common)
	}()

	jsonMessage := <-direct
	if jsonMessage == nil {
		jsonMessage = <-common
	} else {
		//todo: unlock chanel here to close it later
	}

	if jsonMessage != nil {
		jsonMessage.URL = link

		js, err := json.Marshal(jsonMessage)
		if err != nil {
			return []byte{}, err
		}

		return js, nil
	}

	return []byte{}, errors.New("can't create json message")
}

func sendBadResponse(writer http.ResponseWriter, request *http.Request) {
	writer.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(writer, "Bad url request: %s", request.URL)
}

func sendResponse(writer http.ResponseWriter, body []byte) (int, error) {
	writer.Header().Set("Content-Length", strconv.Itoa(len(body)))
	return writer.Write(body)
}

func linkParser(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "application/json")
	writer.Header().Set("Server", "link parser")

	query := request.URL.Query()
	link := query.Get("url")

	if len(link) == 0 {
		sendBadResponse(writer, request)
		return
	}

	if parserDB == nil {
		fmt.Println("\nlinkParser: DB is nil")
	}

	if parserDB != nil {
		cache, err := parserDB.Get([]byte(link), nil)
		if err != leveldb.ErrNotFound {
			log.Println(cache)
			sendResponse(writer, cache)
		} else {
			log.Println(err)

			validURL, err := validateURL(link, nil)
			if err != nil {
				log.Println(err)
			}

			js, err := processLink(validURL)

			parserDB.Put([]byte(validURL), js, nil)

			if err != nil {
				log.Println(err)
			} else {
				fmt.Println(string(js))
			}

			sendResponse(writer, js)
		}
	}
}

func handleCtrlC(c chan os.Signal) {
	sig := <-c

	if parserDB != nil {
		parserDB.Close()
	}

	fmt.Println("\nsignal: ", sig)
	os.Exit(0)
}

// InitParserDB ...
func InitParserDB() {
	tempDir := os.TempDir()
	fmt.Println(tempDir)
	parserDBFile := filepath.Join(tempDir, "parser.db")
	fmt.Println(parserDBFile)

	var err error
	parserDB, err = leveldb.OpenFile(parserDBFile, nil)
	if err != nil {
		log.Println(err)
	}

	//defer parserDB.Close()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go handleCtrlC(c)
}
