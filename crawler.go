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
	Title       string
	Description string
	Keywords    string
	Url         string
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

	var input string
	fmt.Scanln(&input)
}

func crawler(link string) {
	if link[len(link)-1:] == "/" {
		link = link[:len(link)-1]
	}

	result := Link{}
	cLinks.Find(bson.M{"url": link}).One(&result)

	if result.Url == "" {
		resp, err := http.Get(link)
		if err != nil {
			fmt.Printf("Error, %v\n", err)
		} else {
			defer resp.Body.Close()

			regex, _ := regexp.Compile(`text\/html`)
			if regex.MatchString(resp.Header.Get("Content-Type")) {
				fmt.Println(link)

				body, _ := ioutil.ReadAll(resp.Body)
				if body != nil {
					addPage(string(body), link)
					linkChecked(link)
					findLinks(string(body), link)
				}
			}
		}
	}
}

func findLinks(body, baselink string) {
	regex, _ := regexp.Compile(`href=\"(.*?)\"`)
	links := regex.FindAllStringSubmatch(body, -1)

	if len(links) > 0 {
		for _, link := range links {
			if link[1] != baselink && link[1] != "" {
				switch link[1][0:1] {
				case "/":
					if link[1][0:2] == "//" {
						go crawler("http:" + link)
					} else {
						crawler(baselink + link[1])
					}
				case "h":
					go crawler(link[1])
				}
			}
		}
	}
}

func linkChecked(link string) {
	next_date := (time.Now().UnixNano() / int64(time.Second)) + int64(time.Hour*24*7)
	cLinks.Insert(&Link{Url: link, LastVisit: time.Now(), NextCheck: next_date})
}

func addPage(body, link string) {
	regex, _ := regexp.Compile(`\<title\>(.*?)\<\/title\>`)
	tag := regex.FindAllStringSubmatch(body, 1)

	var title string
	if len(title) > 0 {
		title = tag[0][1]
	}

	var webpage Webpage = Webpage{title, "TESTE", "teste", link}

	cPages.Insert(webpage)
}
