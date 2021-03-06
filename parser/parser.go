package parser

import (
	"golang.org/x/net/html"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Link struct {
	Count   int
	Checked bool
	Routine bool
}

var links = make(map[string]Link)

var mutex = &sync.RWMutex{}

var wg = sync.WaitGroup{}

func Parser(uri string) (map[string]Link, error) {
	mutex.Lock()
	links[uri] = Link{Count: 1, Checked: false, Routine: false}
	mutex.Unlock()
	for {
		mutex.RLock()
		li := make(map[string]Link, len(links))
		for key, value := range links {
			li[key] = value
		}
		mutex.RUnlock()
		var checked, runned, run int
		for k, v := range li {
			if v.Checked == false {
				if v.Routine == false {
					mutex.Lock()
					links[k] = Link{Count: links[k].Count, Checked: links[k].Checked, Routine: true}
					mutex.Unlock()
					run++
					wg.Add(1)
					go linksWriter(k, uri)
				} else {
					runned++
				}
			} else {
				checked++
			}
		}
		time.Sleep(time.Second * 1)
		mutex.RLock()
		summ := len(links)
		mutex.RUnlock()
		//fmt.Printf("Total links:\t%v\t\tChecked:\t%v\t\tRunning:\t%v\t\tRuned:\t%v\n", summ, checked, runned, run)
		if summ == checked {
			break
		}
	}
	wg.Wait()
	return links, nil
}

func linksWriter(linka, uri string) {
	defer wg.Done()
	response, err := http.Get(linka)
	if err != nil {
		mutex.Lock()
		links[linka] = Link{Checked: false, Count: links[linka].Count, Routine: false}
		mutex.Unlock()
		return
	}
	defer response.Body.Close()
	if response.StatusCode >= 501 {
		mutex.Lock()
		links[linka] = Link{Checked: false, Count: links[linka].Count, Routine: false}
		mutex.Unlock()
		response.Body.Close()
		return
	}
	if response.StatusCode != http.StatusOK {
		mutex.Lock()
		links[linka] = Link{Checked: true, Count: links[linka].Count, Routine: false}
		mutex.Unlock()
		response.Body.Close()
		return
	}
	if response.StatusCode == 200 {
		if response.Header["Content-Type"][0] == "text/html" {
			tokenizer := html.NewTokenizer(response.Body)
		top:
			for {
				linkInfo := tokenizer.Next()
				switch {
				case linkInfo == html.ErrorToken:
					break top
				case linkInfo == html.StartTagToken:
					tagInfo := tokenizer.Token()
					isAnchor := tagInfo.Data == "a"
					if !isAnchor {
						continue top
					}
					ok, url := getHref(tagInfo, uri)
					if !ok {
						continue top
					}
					mutex.Lock()
					links[url] = Link{Count: links[url].Count + 1, Checked: links[url].Checked, Routine: links[url].Routine}
					mutex.Unlock()
				}
			}
			mutex.Lock()
			links[linka] = Link{Checked: true, Count: links[linka].Count, Routine: false}
			mutex.Unlock()
		} else {

			mutex.Lock()
			delete(links, linka)
			mutex.Unlock()
		}
	}
}

func getHref(t html.Token, uri string) (ok bool, href string) {
	for _, a := range t.Attr {
		if a.Key == "href" {
			if strings.Contains(a.Val, "@") {
				return false, ""
			}
			if a.Val == "#" {
				return false, ""
			}
			href = a.Val
			href = strings.Replace(href, "./", uri, 1)
			href = strings.Replace(href, "www.", "", 1)
			if strings.Contains(href, "#") {
				href = href[:strings.Index(href, "#")]
			}
			if len(href) < 4 {
				return false, ""
			}
			if href[0:4] != "http" {
				href = uri + href
			}
			if !strings.Contains(href, uri) {
				return false, ""
			}
			last4 := href[len(href)-4:]
			if last4 == ".pdf" ||
				last4 == ".doc" ||
				last4 == ".xls" ||
				last4 == ".txt" ||
				last4 == ".rtf" ||
				last4 == ".jpg" ||
				last4 == "jpeg" ||
				last4 == "docx" ||
				last4 == "xlsx" {
				return false, ""
			}
			ok = true
		}
	}
	return ok, href
}
