package main

import (
	"crypto/md5"
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

type Webpage struct {
	Title       string
	Description string
	Keywords    string
	Hash        string
	Url         string
}

type Link struct {
	Url       string
	LastVisit time.Time
	NextCheck int64
}

var cLinks *mgo.Collection
var cPages *mgo.Collection

var domains map[string]string = map[string]string{"www.bis2bis.com.br": "OK"}
var blocks map[string]string = map[string]string{"datatracker.ietf.org": "OK"}

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
		client := &http.Client{}

		req, err := http.NewRequest("GET", link, nil)
		if err != nil {
			fmt.Printf("Error, %v\n", err)
		} else {
			req.Header.Set("User-Agent", "Webcrawler-bot version 1.0")

			resp, err := client.Do(req)
			if err != nil {
				fmt.Printf("Error, %v\n", err)
			} else {
				defer resp.Body.Close()

				regex, _ := regexp.Compile(`text\/html`)
				if regex.MatchString(resp.Header.Get("Content-Type")) {
					fmt.Println(link)

					body, _ := ioutil.ReadAll(resp.Body)
					if body != nil {
						if addPage(string(body), link) {
							linkChecked(link)
							findLinks(string(body), link)
						}
					}
				}
			}
		}
	}
}

func findLinks(body, baselink string) {
	regex, _ := regexp.Compile(`href=\"(.*?)\"`)
	links := regex.FindAllStringSubmatch(body, -1)

	if len(links) > 0 {
		bURL, _ := url.Parse(baselink)
		for _, link := range links {
			if link[1] != baselink {
				u, _ := url.Parse(link[1])
				if _, ok := blocks[u.Host]; u.Host != "" && ok == false {
					if _, ok := domains[u.Host]; ok {
						if u.Scheme == "" {
							if link[1][0:2] == "//" {
								u.Scheme = "http"
								crawler(u.String())
							} else {
								u.Scheme = bURL.Scheme
								u.Host = bURL.Host
								crawler(u.String())
							}
						} else {
							crawler(link[1])
						}
					} else {
						domains[u.Host] = "OK"

						if u.Scheme == "" {
							u.Scheme = "http"
						}

						go crawler(u.String())
					}
				}
			}
		}
	}
}

func linkChecked(link string) {
	next_date := (time.Now().UnixNano() / int64(time.Second)) + int64(time.Hour*24*7)
	cLinks.Insert(&Link{Url: link, LastVisit: time.Now(), NextCheck: next_date})
}

func addPage(body, link string) bool {
	regex, _ := regexp.Compile(`\<title\>(.*?)\<\/title\>`)
	tag := regex.FindAllStringSubmatch(body, 1)

	var title string
	if len(tag) > 0 {
		title = tag[0][1]
	}

	h := md5.New()
	io.WriteString(h, body)

	result := Webpage{}
	cLinks.Find(bson.M{"url": link}).One(&result)

	if result.Url == "" {
		var webpage Webpage = Webpage{title, "TESTE", "teste", string(h.Sum(nil)), link}

		cPages.Insert(webpage)

		return true
	}

	return false
}
