package main

import (
	"bufio"
	"fmt"
	"github.com/mvdan/xurls"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

type Fetcher interface {
	// Fetch 返回 URL 的 body 内容，并且将在这个页面上找到的 URL 放到一个 slice 中。
	Fetch(url string) (body string, urls []string, err error)
}

var lockx = make(chan int, 30)

// 同步通信使用
func LockFun(f func()) {
	lockx <- 1
	f()
	<-lockx
}

var visited map[string]bool = make(map[string]bool)

// Crawl 使用 fetcher 从某个 URL 开始递归的爬取页面，直到达到最大深度。
func Crawl(url string, depth int, fetcher Fetcher, banner chan int) {
	// TODO: 并行的抓取 URL。
	// TODO: 不重复抓取页面。
	// 下面并没有实现上面两种情况：
	if depth <= 0 || visited[url] {
		banner <- 1
		return
	}
	body, urls, err := fetcher.Fetch(url)
	LockFun(func() {
		visited[url] = true
	})
	fmt.Printf("found: %s %q\n", url, body)
	if err != nil {
		fmt.Println(err)
		banner <- 1
		return
	}
	subBanner := make(chan int, len(urls))
	for _, u := range urls {
		//并行
		Crawl(u, depth-1, fetcher, subBanner)
	}
	for i := 0; i < len(urls); i++ {
		<-subBanner
	}
	banner <- 1
	return
}

func main() {
	// 爬行深度
	var length = 2
	// urls 待爬 url
	inputFile, inputError := os.Open("urls.txt")
	if inputError != nil {
		fmt.Println("open urls.txt failed ...")
		return
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	for {
		inputString, readerError := inputReader.ReadString('\n')
		fmt.Println(inputString)
		if readerError == io.EOF {
			return
		}
		mainBanner := make(chan int, 1)
		Crawl(strings.TrimSpace(inputString), length, fetcher, mainBanner)
		<-mainBanner
	}
}

// fakeFetcher 是返回若干结果的 Fetcher。
type fakeFetcher map[string]*fakeResult

type fakeResult struct {
	body string
	urls []string
}

func (f fakeFetcher) Fetch(url string) (string, []string, error) {
	res, err := http.Get(url)
	if err != nil {
		return "test Title", nil, err
	}
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "test Title", nil, err
	}
	urls := xurls.Strict.FindAllString(string(data), -1)
	for _, url := range urls {
		fmt.Printf("Got: %v\n", url)
	}
	body := strings.Split(url, "/")
	return body[2], urls, nil
	//if res, ok := f[url]; ok {
	//return res.body, res.urls, nil
	//}
	//return "", nil, fmt.Errorf("not found: %s", url)
}

// fetcher 是填充后的 fakeFetcher。
var fetcher = fakeFetcher{}
