package main

import (
	"fmt"
	"fuskbreak/pager"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
)

const version string = "1.1"

// Page is the main page struct
type Page struct {
	Url       string     // page url
	Type      string     // video / pictures
	Text      *string    // the html text of the page
	Dom       *html.Node // the page converted to dom
	Title     string     // meta=description content
	Thumbnail string     // meta[property="og:image"] content
	VideoURI  string     // link to the video directly
}

type mapPages map[string]*Page

var subPages mapPages  // Sub Pages
var fifoPages []string // Sub Pages

// PageCreator create a page
func PageCreator(url string) Page {
	//log.Print("Fethcing Page: ", url)

	var Type string
	if strings.Contains(url, "/video/") {
		Type = "video"
	} else if strings.Contains(url, "/pictures/") {
		Type = "pictures"
	}
	// else {
	// 	log.Fatal("Unknown type in url")
	// }

	//log.Println(" - Type: ", Type)

	text, err := pager.GetPage(url)
	if err != nil {
		log.Fatal(err)
	}

	dom, err := html.Parse(strings.NewReader(*text))
	if err != nil {
		log.Fatal(err)
	}

	return Page{Url: url, Type: Type, Text: text, Dom: dom}
}

// GetItems get all items
func (p *Page) GetItems(size int) []string {
	var err error
	var hrefs []string

	// compile selector
	s, err := cascadia.Compile(".Timestream-item a")
	if err != nil {
		log.Fatal(err)
	}

	// get all items
	matches := s.MatchAll(p.Dom)
	//match := s.MatchFirst(doc)

	for _, match := range matches {
		got := pager.NodeString(match)
		//fmt.Println("Results:", got)
		href, err := pager.ParseHref(got)
		if err != nil {
			log.Fatal(err)
		}

		log.Println("HREF:", href)

		hrefs = append(hrefs, href)
	}

	return hrefs
}

// GetInfo get info on the page
func (p *Page) GetInfo() {
	var err error

	// compile selectors
	sDescription, err := cascadia.Compile("meta[name=\"description\"]")
	if err != nil {
		log.Fatal(err)
	}
	match := sDescription.MatchFirst(p.Dom)
	p.Title = *pager.GetAttr(match, "content")

	sImage, err := cascadia.Compile("meta[property=\"og:image\"]")
	if err != nil {
		log.Fatal(err)
	}
	match = sImage.MatchFirst(p.Dom)
	p.Thumbnail = *pager.GetAttr(match, "content")

}

// GetVideo gets the video
func (p *Page) GetVideo() {
	var err error

	// compile selectors
	sDescription, err := cascadia.Compile("meta[name=\"embed_video_url\"]")
	if err != nil {
		log.Fatal(err)
	}
	match := sDescription.MatchFirst(p.Dom)
	embedURL := pager.GetAttr(match, "content")
	if embedURL == nil {
		log.Println("embed_video_url Not found in page")
		return
	}

	embedPage := PageCreator(*embedURL)
	p.VideoURI = pager.StupidJSON("videoUri", *embedPage.Text)
	p.VideoURI = strings.Replace(p.VideoURI, "496_kbps", "864_kbps", 1) // upgrade the kbps

	//log.Println("VideoURI:", p.VideoURI)
}

// GetByType get page by it's type
func (p *Page) GetByType() {
	p.GetInfo()
	switch p.Type {
	case "pictures":
		log.Println("Pictures not implemented yet")
	case "video":
		p.GetVideo()
	default:
		//p.GetItems()
	}
}

// --------------------------------------------------------------------------

func fetcher(url string, pages *mapPages, ch chan string) {
	log.Println("Fetcher: ", url)
	page := PageCreator(url)
	page.GetByType()
	(*pages)[url] = &page
	ch <- url
}

func main() {
	// check for parameters
	if len(os.Args) <= 1 {
		log.Fatal("Usage ", os.Args[0], " [number of posts to fetch]")
	}
	size, err := strconv.Atoi(os.Args[1])
	if size <= 0 || err != nil {
		log.Println("Usage ", os.Args[0], " [number of posts to fetch]")
		log.Fatal("Error: ", err)
	}

	subPages = make(mapPages)

	// fetch main pages
	log.Println("START fuskbreak version:" + version)
	for page := 0; len(fifoPages) < size; page++ {
		nextPage := PageCreator("http://www.break.com/" + strconv.Itoa(page))
		moreItems := nextPage.GetItems(size)
		for _, href := range moreItems { // merging maps and adding to mainPage
			_, ok := subPages[href]
			if !ok {
				subPages[href] = nil
				fifoPages = append(fifoPages, href)
			}

		}
		log.Println("Current items: ", len(fifoPages), " items")
	}

	// start fetching items (using go routines)
	log.Println("Found ", len(fifoPages), " items")

	ch := make(chan string)
	for _, url := range fifoPages {
		go fetcher(url, &subPages, ch)
	}

	// waiting for fetchers to finish
	log.Println("Waiting ...")
	for size = len(fifoPages); size > 0; size-- {
		url := <-ch
		log.Println(size, ". Received: ", url)
	}

	// echo results to the screen
	var str string
	str += "<!doctype html><html><head><title>break.com fuskator</title><meta charset=\"UTF-8\"><style>html {text-align: center} footer {padding: 40px}</style></head><body><h1>break.com fuskator</h1>"

	n := 0
	for _, url := range fifoPages {
		n++
		p, ok := subPages[url]
		if !ok || p == nil {
			continue
		}

		str += "<hr><section><a href=\"" + p.Url + "\" target=\"blank\"><h1>" + strconv.Itoa(n) + ". " + p.Title + " (" + p.Type + ") </a></h1>"
		if p.VideoURI != "" {
			str += "<br><a href=\"" + p.VideoURI + "\" target=\"blank\"><img src=\"" + p.Thumbnail + "\" style=\"width:25%\"></a>"
		} else {
			str += "<br><a href=\"" + p.Url + "\" target=\"blank\"><img src=\"" + p.Thumbnail + "\" style=\"width:25%\"></a>"
		}
		str += "</section>"
	}

	str += "<br><br><footer>Made by fuskbreak (Cnaan Aviv)</footer></body></html>"

	fmt.Println(str)

	log.Println("END")
}
