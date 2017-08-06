package main

import (
	"fmt"
	"fuskbreak/pager"
	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"log"
	"os"
	"strconv"
	"strings"
)

const version string = "1.0"

// --------------------------------------------------------------------------
type Page struct {
	Url       string           // page url
	Type      string           // video / pictures
	Text      *string          // the html text of the page
	Dom       *html.Node       // the page converted to dom
	Title     string           // meta=description content
	Thumbnail string           // meta[property="og:image"] content
	VideoUri  string           // link to the video directly
	SubPages  map[string]*Page // Sub Pages
	FifoPages []*string        // Sub Pages
}

// create a page
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

	return Page{Url: url, Type: Type, Text: text, Dom: dom, SubPages: make(map[string]*Page)}
}

// get all items
func (p *Page) GetItems(size int) map[string]*Page {
	var err error = nil

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

		_, ok := p.SubPages[href]
		if !ok {
			p.SubPages[href] = nil
			p.FifoPages = append(p.FifoPages, &href)
			size--
			if size == 0 {
				break
			}
		}
	}

	return p.SubPages
}

func (p *Page) GetInfo() {
	var err error = nil

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

func (p *Page) GetVideo() {
	var err error = nil

	// compile selectors
	sDescription, err := cascadia.Compile("meta[name=\"embed_video_url\"]")
	if err != nil {
		log.Fatal(err)
	}
	match := sDescription.MatchFirst(p.Dom)
	embedUrl := pager.GetAttr(match, "content")
	if embedUrl == nil {
		log.Println("embed_video_url Not found in page")
		return
	}

	embedPage := PageCreator(*embedUrl)
	p.VideoUri = pager.StupidJson("videoUri", *embedPage.Text)

	//log.Println("videoUri:", p.VideoUri)
}

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

func (p *Page) AddSubPage(subPage *Page) {
	p.SubPages[subPage.Url] = subPage
}

func (p *Page) GetHTML(n int) string {
	var str string

	if p.Type == "" {
		str += "<!doctype html><html><head><title>break.com fuskator</title><meta charset=\"UTF-8\"><style>html {text-align: center} footer {padding: 40px}</style></head><body><h1>break.com fuskator</h1>"
	}

	str += "<hr><section><a href=\"" + p.Url + "\" target=\"blank\"><h1>" + strconv.Itoa(n) + ". " + p.Title + "</a></h1>"

	if p.VideoUri != "" {
		str += "<br><a href=\"" + p.VideoUri + "\" target=\"blank\"><img src=\"" + p.Thumbnail + "\" style=\"width:25%\"></a>"
	} else {
		str += "<br><a href=\"" + p.Url + "\" target=\"blank\"><img src=\"" + p.Thumbnail + "\" style=\"width:25%\"></a>"
	}

	str += "</section>"

	n = 0
	for _, href := range p.FifoPages {
		subPage, ok := p.SubPages[*href]
		if subPage != nil && ok {
			str += subPage.GetHTML(n)
			n++
		}
	}

	if p.Type == "" {
		str += "<br><br><footer>Made by fuskbreak (Cnaan Aviv)</footer></body></html>"
	}
	return str
}

// --------------------------------------------------------------------------

func fetcher(url string, mainPage *Page, ch chan string) {
	log.Println("Fetcher: ", url)
	page := PageCreator(url)
	page.GetByType()
	mainPage.AddSubPage(&page)
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

	// fetch main pages
	log.Println("START fuskbreak version:" + version)
	mainPage := PageCreator("http://www.break.com")
	items := mainPage.GetItems(size)
	//fmt.Println("Found Items", len(*items))

	for page := 1; len(items) < size; page++ {
		nextPage := PageCreator("http://www.break.com/" + strconv.Itoa(page))
		for k, v := range nextPage.GetItems(size) { // merging maps
			items[k] = v
		}
	}

	// start fetching items (using go routines)
	log.Println("Found ", len(items), " items")
	ch := make(chan string)
	for url, _ := range items {
		go fetcher(url, &mainPage, ch)
	}

	// waiting for fetchers to finish
	log.Println("Waiting ...")
	for ; size > 0; size-- {
		url := <-ch
		log.Println(size, ". Received: ", url)
	}

	// echo results to the screen
	fmt.Println(mainPage.GetHTML(0))

	log.Println("END")
}
