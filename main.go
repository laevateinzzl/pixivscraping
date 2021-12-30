package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"time"

	"github.com/gocolly/colly"
	"github.com/tidwall/gjson"
	"github.com/valyala/fasthttp"
)

var ua string = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.93 Safari/537.36"
var cookie string = ""

func main() {
	// uid := 6657532
	// uid := 21160127
	// uid := 6890797
	uid := 32548944

	start := time.Now()

	res := geturl(int64(uid))

	ch := make(chan string)

	fileList := getpicture(res)

	for _, url := range fileList {
		go downpicture(url, ch)
	}

	for range fileList {
		fmt.Println(<-ch) // receive from channel ch

	}
	fmt.Printf("%.2fs elapsed\n", time.Since(start).Seconds())

}

func geturl(uid int64) []string {
	var idList []string

	url := fmt.Sprintf("https://www.pixiv.net/ajax/user/%d/profile/all?lang=zh", uid)

	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	buf := resp.Body

	data, _ := io.ReadAll(buf)
	jsonData := string(data)

	res := gjson.Get(jsonData, "body.illusts").Map()

	for id, _ := range res {
		idList = append(idList, id)
	}

	return idList
}

func getpicture(urlList []string) []string {

	c := colly.NewCollector(
		colly.UserAgent(ua),
	)

	var plist []string

	for _, u := range urlList {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set("referer", fmt.Sprintf("https://www.pixiv.net/artworks/%s", u))
			r.Headers.Set("cookie", cookie)
		})

		c.OnResponse(func(r *colly.Response) {
			jsonData := string(r.Body)
			u := gjson.Get(jsonData, "body").Array()
			for _, v := range u {
				url := gjson.Get(v.Raw, "urls.original").String()
				plist = append(plist, url)
			}
		})

		c.Visit(fmt.Sprintf("https://www.pixiv.net/ajax/illust/%s/pages?lang=zh", u))
	}

	return removeDuplicateElement(plist)
}

func downpicture(url string, ch chan<- string) {
	start := time.Now()
	// client := &http.Client{}
	client := &fasthttp.Client{}
	// req, _ := http.NewRequest("GET", url, nil)
	var req fasthttp.Request
	req.SetRequestURI(url)
	req.Header.Add("referer", "https://www.pixiv.net/")
	req.Header.Add("user-agnet", ua)
	var resp fasthttp.Response

	err := client.Do(&req, &resp)
	if err != nil {
		ch <- fmt.Sprintf("while reading %s: %v", url, err)
		return
	}

	reg, _ := regexp.Compile(`([^\\/]+)\.(jpg|png)`)
	name := reg.FindStringSubmatch(url)[0]
	fmt.Println(name)
	out, _ := os.Create("avatar/" + name)

	data := bytes.NewReader(resp.Body())

	size, err := io.Copy(out, data)
	if err != nil {
		ch <- fmt.Sprintf("while reading %s: %v", url, err)
		return
	}
	secs := time.Since(start).Seconds()
	ch <- fmt.Sprintf("%.2fs  %7d  %s", secs, size, url)

}

func removeDuplicateElement(languages []string) []string {
	result := make([]string, 0, len(languages))
	temp := map[string]struct{}{}
	for _, item := range languages {
		if _, ok := temp[item]; !ok {
			temp[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
