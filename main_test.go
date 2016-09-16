// main_test.go
package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"testing"
)

var testLinks = [...]string{
	"https://www.youtube.com/",
	"https://vimeo.com",
	"https://soundcloud.com",
	"http://bandcamp.com/",
	"https://www.yahoo.com/",
	"http://www.amazon.com/",
	"https://en.wikipedia.org",
	"https://twitter.com",
	"http://www.boredpanda.com/",
	"http://lifehacker.com/",
	"http://www.ebay.com/",
	"https://www.facebook.com",
	"http://edition.cnn.com/",
	"http://www.bbc.com/",
	"http://rutube.ru/",
	"https://yandex.by",
	"http://techcrunch.com/",
	"http://www.dailymotion.com/",
	"http://espn.go.com/",
	"http://coub.com/",
	"http://tech.onliner.by/",
	"http://vk.com/",
	"https://www.tumblr.com",
	"https://instagram.com",
	"https://medium.com",
	"https://plus.google.com/",
	"http://habrahabr.ru/",
	"http://www.last.fm/",
	"https://www.flickr.com",
	"http://www.ted.com/",
	"https://www.pinterest.com",
	"http://www.deviantart.com/",
	"http://www.taobao.com/",
	"http://imgur.com/",
	"http://www.alibaba.com/",
	"http://www.aliexpress.com/",
	"http://www.rakuten.co.jp/",
	"https://www.blogger.com",
	"http://www.buzzfeed.com/",
	"http://www.usatoday.com",
	"http://www.forbes.com/",
	"http://www.imdb.com/",
	"http://www.kinopoisk.ru/",
	"https://500px.com",
	"http://www.latimes.com/",
	"http://thepatternedplate.wordpress.com",
	"http://www.hulu.com/",
	"https://www.spotify.com",
	"http://письмо.рф",
	"http://письмо.Рф",
	"http://письмо.рф/more",
	"http://минск.квеструм.бел"}

func TestMain(t *testing.T) {
	for _, val := range testLinks {
		log.SetOutput(ioutil.Discard)
		t.Logf("Running test for %s\n", val)

		validUrl, err := validateUrl(val, nil)
		if err != nil {
			t.Error(err)
		}

		js, err := processLink(validUrl, *originText)

		if err != nil {
			t.Error("Test Error: " + err.Error())
		} else {
			log.Println(js)
			var jsonMessage JsonMessage
			json.Unmarshal(js, &jsonMessage)

			if jsonMessage.Title == "" && jsonMessage.ThumbnailURL == "" {
				t.Error("Test Error: No title and thumbnail")
			} else if len(jsonMessage.Title) > 0 && jsonMessage.ThumbnailURL == "" {
				t.Log("Test Warning: No thumbnail")
			} else {
				t.Log("Test OK")
			}
		}
	}
}

func BenchmarkMain(t *testing.B) {
	js, err := processLink("http://www.bbc.com/", "origin text")
	if err != nil {
		t.Error("Test Error: " + err.Error())
	} else {
		log.Println(string(js))
	}
}
