// CommonProcessingLink.go
package main

import (
	"html"
	"log"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func getContentLength(link *string) int {
	if *link == "" {
		log.Println("thumbnail url is empty")

		return 0
	}

	header := map[string]string{"Accept-Encoding": "*"}
	resp, err := networkRequest(*link, header, link)
	if err != nil {
		return 0
	}

	defer resp.Body.Close()

	res, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		log.Println(err)

		return 0
	} else {
		return res
	}
}

func metaContent(doc *goquery.Document, sel string) string {
	if sel == "" {
		return sel
	}

	node := doc.Find(sel)
	if len(node.Nodes) == 0 {
		return ""
	}

	val, exists := node.Attr("content")
	if !exists {
		return ""
	} else {
		return val
	}
}

func ogTypeToMediaType(ogType string) string {
	if ogType == "" {
		return ""
	}

	videoTypeList := []string{"video", "player", "coub", "tv", "movie", "episode"}
	for _, currType := range videoTypeList {
		if strings.Contains(ogType, currType) {
			return "video"
		}
	}

	audioTypeList := []string{"music", "audio", "song", "album", "band", "playlist", "radio", "sound"}
	for _, currType := range audioTypeList {
		if strings.Contains(ogType, currType) {
			return "audio"
		}
	}

	return "default"
}

func getThumbnailFromBody(doc *goquery.Document, baseUrl *url.URL) (string, int) {
	thumbnailLinks := map[chan int]string{}

	handler := func(index int, img *goquery.Selection) bool {
		attrs := []string{"src", "data-baseurl", "data-img", "load_image", "href", "data-src"}
		var src string

		for _, attr := range attrs {
			var exists bool
			src, exists = img.Attr(attr)

			checkExt := func() bool {
				imageExtList := []string{".png", ".jpeg", ".jpg", ".bmp", ".webp", ".wbmp", ".ico"}

				//specific case: in google search img contains link without ext
				if path.Ext(src) == "" && img.Is("img") {
					return true
				}

				for _, ext := range imageExtList {
					if strings.HasPrefix(path.Ext(src), ext) {
						return true
					}
				}

				return false
			}()

			if !exists || !checkExt {
				continue
			}

			widthHeightRatio := func() float64 {
				width, existH := img.Attr("width")
				height, existW := img.Attr("height")
				if existW && existH {
					w, _ := strconv.ParseFloat(width, 32)
					h, _ := strconv.ParseFloat(height, 32)

					if h == 0 {
						return 0
					}

					return w / h
				}

				return 0
			}()

			if widthHeightRatio > 0 && (widthHeightRatio < 0.5 || widthHeightRatio > 5.0) {
				continue
			}

			src, err := validateURL(src, baseUrl)
			if err != nil {
				continue
			}

			if len(thumbnailLinks) > 10 {
				return false
			}

			length := make(chan int)

			thumbnailLinks[length] = src

			go func() {
				length <- getContentLength(&src)
			}()
		}

		return true
	}

	doc.Find("img").EachWithBreak(handler)
	if len(thumbnailLinks) == 0 {
		doc.Find("link").EachWithBreak(handler)
	}

	//find thumbnail with max size
	temp := map[int]string{}
	var keys []int
	for key, val := range thumbnailLinks {
		length := <-key
		if length == 0 {
			continue
		}

		temp[length] = val
		keys = append(keys, length)

		defer close(key)
	}

	sort.Ints(keys)

	if len(keys) > 0 {
		length := keys[len(keys)-1]

		return temp[length], length
	} else {
		return "", 0
	}
}

func getCharset(contentType string, doc *goquery.Document) string {
	fromContentType := func(contentType string) string {
		contentType = strings.ToLower(contentType)

		arr := strings.Split(contentType, ";")
		if len(arr) == 2 {
			charset := strings.Replace(arr[1], "charset=", "", -1)

			return strings.TrimSpace(charset)
		}

		return ""
	}

	charset := fromContentType(contentType)
	if charset != "" {
		return charset
	}

	meta := doc.Find("meta[charset]")
	charset, _ = meta.Attr("charset")
	if charset != "" {
		return charset
	}

	meta = doc.Find(`meta[content^="text/html"]`)
	contentType, _ = meta.Attr("content")

	return fromContentType(contentType)
}

func commonLinkProcessing(link string) *JSONMessage {
	resp, err := networkRequest(link, nil, nil)
	if err != nil {
		return nil
	}

	defer resp.Body.Close()

	//creating doc
	doc, err := goquery.NewDocumentFromResponse(resp)
	if err != nil {
		log.Println(err)

		return nil
	}

	//getting type
	mediaType := ogTypeToMediaType(metaContent(doc, `meta[property="og:type"]`))
	if mediaType == "" {
		mediaType = ogTypeToMediaType(metaContent(doc, `meta[name="twitter:card"]`))
		if mediaType == "" {
			mediaType = "default"
		}
	}

	//getting title
	title := metaContent(doc, `meta[property="og:title"]`)
	if title == "" {
		title = metaContent(doc, `meta[property="twitter:title"]`)
		if title == "" {
			title = doc.Find("title").First().Text()
			if title == "" {
				title = metaContent(doc, `meta[property="og:description"]`)
				if title == "" {
					title = metaContent(doc, `meta[property="twitter:description"]`)
					if title == "" {
						title = doc.Find("description").First().Text()
					}
				}
			}
		}
	}

	//getting thumbnail
	thumbnail := metaContent(doc, `meta[property="og:image"]`)
	if thumbnail == "" {
		thumbnail = metaContent(doc, `meta[property="twitter:image"]`)
	}

	var contentLength int
	if thumbnail, err = validateURL(thumbnail, resp.Request.URL); err == nil {
		contentLength = getContentLength(&thumbnail)
	}

	if err != nil || contentLength == 0 {
		thumbnail, contentLength = getThumbnailFromBody(doc, resp.Request.URL)
	}

	//getting charset
	charset := getCharset(resp.Header.Get("Content-Type"), doc)

	//craeate json message
	jsonMessage := JSONMessage{}
	jsonMessage.Title = html.UnescapeString(toUtf8(strings.TrimSpace(title), charset))
	jsonMessage.ThumbnailURL = thumbnail
	jsonMessage.ContentType = resp.Header.Get("Content-Type")
	jsonMessage.ContentLength = contentLength
	jsonMessage.Type = mediaType

	return &jsonMessage
}
