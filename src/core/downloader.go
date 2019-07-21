package core

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

type Downloader struct {
	Url             string
	ThreadCount     int
	RealThreadCount int
	FilePath        string
	ContentLength   int64
	Speed           float32
	SegmentSize     int64
	Proxies         string
	pool            *CoroutinePool
	httpClient      http.Client
	downloadedSize  int64
}

var DefaultDownloader = &Downloader{
	ThreadCount: 10,
	SegmentSize: 1024 * 1024 * 2,
}

type downloadSegment struct {
	url      string
	filepath string
	startPos int64
	length   int64
}

func (downloader *Downloader) Download(url, filepath string) error {
	downloader.Url = url
	downloader.FilePath = filepath

	// 1. 获取文件长度
	_ = downloader.getContentLength(url)
	fmt.Println("Content-Length: ", downloader.ContentLength)

	// 2. 计算线程数量
	downloader.RealThreadCount = downloader.calculateThreadNum()
	fmt.Println("download thread count: ", downloader.RealThreadCount)

	// 4. 创建下载协程
	downloader.createThreadPool(downloader.RealThreadCount)

	// 5. 创建下载分段任务
	go downloader.patchSegment()
	go downloader.ShowDownloadSpeed()
	DestroyCoroutinePool(downloader.pool)
	return nil
}

func (downloader *Downloader) calculateThreadNum() int {
	count := int(downloader.ContentLength / downloader.SegmentSize)
	if count > downloader.ThreadCount {
		return downloader.ThreadCount
	}
	return count
}

func (downloader *Downloader) getContentLength(url string) error {
	request, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		log.Println(err)
		return err
	}

	request.Header.Add("Referer", strings.Split(downloader.Url, "/")[2])
	request.Header.Add("Host", strings.Split(downloader.Url, "/")[2])
	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36")

	resp, err := downloader.httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return err
	}
	downloader.ContentLength, _ = strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	return nil
}

func (downloader *Downloader) createThreadPool(threadNum int) {
	downloader.pool = CreateCoroutinePool(threadNum, func(data interface{}) {
		formatData := data.(*downloadSegment)
		content, err := downloader.downloadContent(formatData.startPos, formatData.length)
		if err != nil {
			fmt.Println(err)
			return
		}
		atomic.AddInt64(&downloader.downloadedSize, int64(len(content)))
		if err := downloader.writeDataToFile(formatData.filepath, formatData.startPos, content); err != nil {
			fmt.Println(err)
			return
		}
	})
}

func (downloader *Downloader) patchSegment() {
	var i int64 = 0
	for i = 0; i < downloader.ContentLength; i += downloader.SegmentSize {
		length := downloader.SegmentSize
		if downloader.ContentLength-i < downloader.SegmentSize {
			length = downloader.ContentLength - i + 1
		}
		downloader.pool.AddTask(&downloadSegment{url: downloader.Url, filepath: downloader.FilePath, startPos: i, length: length})
	}
}

func (downloader *Downloader) downloadContent(startPos, length int64) ([]byte, error) {
	var content []byte
	request, err := http.NewRequest("GET", downloader.Url, nil)
	if err != nil {
		log.Println(err)
		return content, nil
	}

	request.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Safari/537.36")
	request.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", startPos, startPos+length-1))
	request.Header.Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", startPos, startPos+length-1, downloader.ContentLength))
	request.Header.Add("Content-Length", fmt.Sprintf("%d", length))
	request.Header.Add("Referer", strings.Split(downloader.Url, "/")[2])
	request.Header.Add("Host", strings.Split(downloader.Url, "/")[2])

	resp, err := downloader.httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return content, err
	}
	defer resp.Body.Close()
	content, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return content, err
	}
	return content, nil
}

func (downloader *Downloader) writeDataToFile(filepath string, offset int64, data []byte) error {
	file, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE, 0777)
	if err != nil {
		log.Println(err)
		return err
	}

	writenLen, err := file.WriteAt(data, offset)
	if err != nil || writenLen != len(data) {
		return err
	}
	return nil
}

func (downloader *Downloader) ShowDownloadSpeed() {
	currentSize := downloader.downloadedSize
	for {
		time.Sleep(time.Second)
		now := downloader.downloadedSize
		size := now - currentSize
		currentSize = now
		if (size >> 30) > 0 {
			fmt.Printf("download speed: %d GB/s\n", size>>30)
		} else if (size >> 20) > 0 {
			fmt.Printf("download speed: %d MB/s\n", size>>20)
		} else if (size >> 10) > 0 {
			fmt.Printf("download speed: %d KB/s\n", size>>10)
		} else {
			fmt.Printf("download speed: %d Byte/s\n", size)
		}
	}
}
