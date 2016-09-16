// decoder.go
package main

import (
	"io/ioutil"
	"log"
	"strings"

	"golang.org/x/net/html/charset"
	"golang.org/x/text/encoding"
	"golang.org/x/text/transform"
)

func toUtf8(str, coding string) string {
	e, _ := charset.Lookup(coding)

	return toUtf8Private(str, e)
}

func toUtf8Private(str string, coding encoding.Encoding) string {
	if coding == nil {
		return str
	}

	sReader := strings.NewReader(str)
	tReader := transform.NewReader(sReader, coding.NewDecoder())
	buf, err := ioutil.ReadAll(tReader)
	if err != err {
		log.Println(err)

		return str
	}

	return string(buf)
}
