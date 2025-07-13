package find

import (
	"github.com/Seven1an/WafReport/Config"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

func FetchNewIDs(baseURL, staticPath string) []string {
	baseURL = strings.TrimRight(baseURL, "/")
	fullFindURL := fmt.Sprintf("%s/log/websec/find", baseURL)

	params := url.Values{}
	params.Set("pageNo", "0")
	params.Set("_", fmt.Sprintf("%d", time.Now().UnixMilli()))
	fullURL := fmt.Sprintf("%s?%s", fullFindURL, params.Encode())

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		log.Fatalf("解析 baseURL 失败: %v", err)
	}
	host := parsedURL.Host

	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Fatalf("创建请求失败: %v", err)
	}

	req.Header.Set("Host", host)
	req.Header.Set("Cookie", Config.Cookie)
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Set("Accept", "text/javascript, text/html, application/xml, text/xml, */*")
	req.Header.Set("X-Prototype-Version", "1.6.0.3")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Origin", baseURL)
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", baseURL+"/log/websec")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("请求发送失败: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("读取响应失败: %v", err)
	}

	re := regexp.MustCompile(`addRisklevelStrategy\('(\d+)'\)`)
	matches := re.FindAllStringSubmatch(string(body), -1)

	var currentIDs []string
	for _, match := range matches {
		currentIDs = append(currentIDs, match[1])
	}

	lastPath := staticPath + "/last_ids.txt"
	newPath := staticPath + "/new_ids.txt"

	lastIDs := loadLines(lastPath)
	lastIDSet := make(map[string]bool)
	for _, id := range lastIDs {
		lastIDSet[id] = true
	}

	var newIDs []string
	for _, id := range currentIDs {
		if !lastIDSet[id] {
			newIDs = append(newIDs, id)
		}
	}

	writeLines(currentIDs, lastPath)

	if len(newIDs) > 0 {
		writeLines(newIDs, newPath)
		fmt.Printf("发现新增 ID %d 个，已写入 %s\n", len(newIDs), newPath)
	} else {
		//fmt.Println("null ID")
	}

	return newIDs
}

func loadLines(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	lines := strings.Split(string(data), "\n")
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}

func writeLines(lines []string, path string) {
	content := strings.Join(lines, "\n") + "\n"
	os.MkdirAll(getDir(path), 0755)
	os.WriteFile(path, []byte(content), 0644)
}

func getDir(path string) string {
	if idx := strings.LastIndex(path, "/"); idx != -1 {
		return path[:idx]
	}
	return "."
}
