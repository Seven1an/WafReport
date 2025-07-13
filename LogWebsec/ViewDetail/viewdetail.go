package viewdetail

import (
	"github.com/Seven1an/WafReport/Bot"
	"github.com/Seven1an/WafReport/Config"
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
)

func ViewDetail(baseURL, staticPath string) {
	newIDsPath := fmt.Sprintf("%s/new_ids.txt", staticPath)
	ids, err := readLines(newIDsPath)
	if err != nil {
		log.Fatalf("读取 %s 失败: %v", newIDsPath, err)
	}
	QueryDetails(ids, baseURL, staticPath)
}

func QueryDetails(ids []string, baseURL, staticPath string) {
	if len(ids) == 0 {
		return
	}

	visitedIPs := make(map[string]bool)
	baseURL = strings.TrimRight(baseURL, "/")

	for _, id := range ids {
		url := fmt.Sprintf("%s/log/websec/viewDetail?id=%s", baseURL, id)
		html, err := fetchHTML(url, baseURL)
		if err != nil {
			log.Printf("查询ID %s 失败: %v", id, err)
			continue
		}

		clientIP := extractField(html, `客户端IP</td>\s*<td>([^<]+)</td>`)
		if clientIP != "" {
			if idx := strings.Index(clientIP, "("); idx != -1 {
				clientIP = clientIP[:idx]
			}
			clientIP = strings.TrimSpace(clientIP)
		}
		if visitedIPs[clientIP] {
			continue
		}
		visitedIPs[clientIP] = true

		var msg strings.Builder
		msg.WriteString("---------------------------------------------\n")
		msg.WriteString("发现攻击IP\n")
		msg.WriteString(clientIP + "\n")
		msg.WriteString("建议封禁\n\n")
		msg.WriteString("捕获设备 【绿盟WAF】\n")

		alertType := extractField(html, `告警类型</td>\s*<td>([^<]+)</td>`)
		if alertType != "" {
			msg.WriteString(fmt.Sprintf("鉴定结果 【%s】\n", strings.TrimSpace(alertType)))
		}

		targetIP := extractField(html, `服务器IP</td>\s*<td>([^<]+)</td>`)
		if targetIP != "" {
			msg.WriteString(fmt.Sprintf("攻击目标 【%s】\n", strings.TrimSpace(targetIP)))
		}

		threatLevel := extractField(html, `风险级别</td>\s*<td><img [^>]*title="([^"]+)"`)
		if threatLevel != "" {
			msg.WriteString(fmt.Sprintf("威胁等级 【%s危】\n", strings.TrimSpace(threatLevel)))
		}

		msg.WriteString("---------------------------------------------")

		// 同步输出终端和机器人
		text := msg.String()
		fmt.Println(text)
		Bot.SendToWeCom(text)
	}
}

func fetchHTML(urlStr string, baseURL string) (string, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	host := parsedURL.Host

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Host", host)
	req.Header.Set("Cookie", Config.Cookie)
	req.Header.Set("Sec-Ch-Ua", `"Chromium";v="104", " Not A;Brand";v="99", "Google Chrome";v="104"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
	req.Header.Set("Upgrade-Insecure-Requests", "1")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "navigate")
	req.Header.Set("Sec-Fetch-Dest", "iframe")
	req.Header.Set("Referer", baseURL+"/")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Connection", "keep-alive")

	client := &http.Client{
		Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func extractField(html, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(html)
	if len(match) >= 2 {
		return match[1]
	}
	return ""
}

func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var result []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			result = append(result, line)
		}
	}
	return result, scanner.Err()
}
