package pager

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

// GetPage fetch one page and parse it as html
func GetPage(url string) (*string, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	str := string(body)
	return &str, nil
}

// GetPageDom fetch one page and parse it as html
func GetPageDom(url string) (*html.Node, error) {
	page, err := GetPage(url)
	if err != nil {
		return nil, err
	}

	return html.Parse(strings.NewReader(*page))
}

// ParseHref Find the fisrt href attribute and return it's value with-in a string
func ParseHref(str string) (string, error) {
	inx := strings.Index(str, "href=\"")
	if inx < 0 {
		return "", errors.New("Not found")
	}
	str = str[inx+6:]
	end := strings.Index(str, "\"")
	if end < 0 {
		return "", errors.New("Not found")
	}
	return string(str[:end]), nil
}

// StupidJSON extract video link from json in a stupid way
func StupidJSON(key string, str string) string {
	//log.Print(" -- video -- ", str)
	key = fmt.Sprintf("\"%v\":", key)
	inx := strings.Index(str, key)
	if inx < 0 {
		return ""
	}
	str = str[inx+len(key)+1:]
	start := strings.Index(str, "\"")
	if start < 0 {
		return ""
	}
	end := strings.Index(str[start+1:], "\"")
	if end <= 0 {
		return ""
	}
	return str[start+1 : end+1]
}

// -----------------------------------------

// GetAttr - finds attribute
func GetAttr(n *html.Node, key string) *string {
	if n != nil {
		for _, attr := range n.Attr {
			if attr.Key == key {
				return &attr.Val
			}
		}
	}
	return nil
}

// NodeString cascadia helper for converting a html.Node to a string
func NodeString(n *html.Node) string {
	buf := bytes.NewBufferString("")
	html.Render(buf, n)
	return buf.String()
}

// CascadiaWrapper cascadia wrappers
func CascadiaWrapper(doc *string, selector string) (*[]string, error) {
	html, err := html.Parse(strings.NewReader(*doc))
	if err != nil {
		log.Fatal(err)
	}

	// compile selector
	s, err := cascadia.Compile(selector)
	if err != nil {
		log.Fatal(err)
	}

	matches := s.MatchAll(html)
	var rvs []string
	for _, match := range matches {
		rvs = append(rvs, NodeString(match))
		fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")
	}

	return &rvs, nil
}
