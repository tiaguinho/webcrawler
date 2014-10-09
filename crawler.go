package main

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
	"regexp"
	"time"
)

type Webpage struct {
	Title string
	Body  string
	Url   string
	Score int
}

type Link struct {
	Url       string
	LastVisit time.Time
	NextCheck int64
}

var cLinks *mgo.Collection
var cPages *mgo.Collection

func main() {
	mgo, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}

	cPages = mgo.DB("webcrawler").C("webpages")
	cLinks = mgo.DB("webcrawler").C("links")

	crawler("http://www.bis2bis.com.br")
}

func crawler(link string) {
	if link[len(link)-1:] == "/" {
		link = link[:len(link)-1]
	}

	result := Link{}
	cLinks.Find(bson.M{"url": link}).One(&result)

	if result.Url == "" {
		fmt.Println(link)
		resp, err := http.Get(link)
		if err != nil {
			fmt.Printf("Error, %v\n", err)
		}
		defer resp.Body.Close()

		regex, _ := regexp.Compile(`text\/html`)
		if regex.MatchString(resp.Header.Get("Content-Type")) {
			body, _ := ioutil.ReadAll(resp.Body)
			if body != nil {
				addPage(resp, string(body), link)
				linkChecked(link)
				findLinks(string(body), link)
			}
		}
	}
}

func findLinks(body, baselink string) {
	regex, _ := regexp.Compile(`href=\"(.*?)\"`)
	links := regex.FindAllStringSubmatch(body, -1)

	if len(links) > 0 {
		for _, link := range links {
			if link[1] != baselink {
				switch link[1][0:1] {
				case "/":
					crawler(baselink + link[1])
				case "h":
					crawler(link[1])
				}
			}
		}
	}
}

func linkChecked(link string) {
	next_date := (time.Now().UnixNano() / int64(time.Second)) + int64(time.Hour*24*7)
	cLinks.Insert(&Link{Url: link, LastVisit: time.Now(), NextCheck: next_date})
}

func addPage(resp *http.Response, body, link string) {
	regex, _ := regexp.Compile(`\<title\>(.*?)\<\/title\>`)
	title := regex.FindAllStringSubmatch(body, 1)

	var webpage Webpage = Webpage{
		Title: title[0][1],
		Body:  body,
		Url:   link,
		Score: 0,
	}

	cPages.Insert(webpage)
}
