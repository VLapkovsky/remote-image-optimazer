// directLinkProcessing.go
package main

import (
	"net/url"
	"path"
	"strconv"
	"strings"
)

func directLinkProcessing(link string) *JSONMessage {
	header := map[string]string{"Accept-Encoding": "*"}
	resp, err := networkRequest(link, header, nil)
	if err != nil {
		return nil
	}

	resp.Body.Close()

	contentType := strings.ToLower(resp.Header.Get("Content-Type"))

	defineFormat := func(mediaType string, extList []string) (string, bool) {
		if strings.HasPrefix(contentType, mediaType+"/") {
			return mediaType, true
		}

		if strings.HasPrefix(contentType, "application/") || contentType == "" {
			for _, ext := range extList {
				if path.Ext(link) == ext {
					return mediaType, true
				}
			}

			return "default", true
		}

		return "default", false
	}

	imageExtList := []string{".png", ".jpeg", ".jpg", ".gif", ".bmp", ".webp", ".wbmp", ".ico"}
	videoExtList := []string{".mp4", ".m4v", ".3gpp", ".3gpp2", ".wmv", ".asf", ".mkv", "mp2ts", "avi", "webm"}
	audioExtList := []string{".mp3", ".m4a", ".wav", ".amr", ".awb", ".ogg", ".acc", "mka", "flac"}

	extList := map[string][]string{"image": imageExtList, "video": videoExtList, "audio": audioExtList}
	for key, val := range extList {
		mediaType, directLink := defineFormat(key, val)

		if directLink {
			jsonMessage := JSONMessage{}
			jsonMessage.Type = mediaType
			jsonMessage.Title = path.Base(link)

			//from percent encoding
			if title, err := url.QueryUnescape(jsonMessage.Title); err == nil {
				jsonMessage.Title = title
			}

			jsonMessage.ContentType = contentType
			jsonMessage.ContentLength, _ = strconv.Atoi(resp.Header.Get("Content-Length"))
			if mediaType == "image" {
				jsonMessage.ThumbnailURL = link
			}

			return &jsonMessage
		}
	}

	return nil
}
