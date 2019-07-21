package main

import (
	"downloader/src/core"
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"
	"strings"
	"time"
)

var (
	url = flag.String("url", "", "download file url")
	filename = flag.String("name", "", "file name to save downloaded file")
)

func main() {
	flag.Parse()
	if *url == "" {
		log.Fatal("you must specify download url.")
		os.Exit(0)
	}

	if *filename == "" {
		strs := strings.Split(*url, "/")
		*filename = strs[len(strs)-1]
	}

	before := time.Now()
	downloader := core.DefaultDownloader
	downloader.Download(*url, *filename)
	duration := time.Now().Sub(before)
	fmt.Println("download finish, cost time: ", duration)
}
