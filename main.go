package main

import (
	"github.com/Seven1an/WafReport/Config"
	"github.com/Seven1an/WafReport/LogWebsec/find"
	"github.com/Seven1an/WafReport/LogWebsec/viewdetail"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func main() {
	fmt.Println("【WafReport】 - By Seven1an")
	// 解析命令行参数
	baseURL := flag.String("u", "", "目标 URL，例如 https://x.x.x.x")
	cookie := flag.String("c", "", "Cookie 内容")
	interval := flag.Int("t", 1, "轮询间隔时间（分钟），最小值1")
	flag.Parse()

	if *baseURL == "" || *cookie == "" {
		fmt.Println("用法示例：WafReport.exe -u https://192.168.x.x -c \"cookie\" -t 多少分钟获取一次")
		os.Exit(1)
	}

	// 限制最小间隔为1分钟
	if *interval < 1 {
		fmt.Println("轮询间隔时间不能小于1分钟，自动调整为1分钟")
		*interval = 1
	}

	// 配置全局变量
	Config.BaseURL = strings.TrimRight(*baseURL, "/")
	Config.Cookie = *cookie

	// static 文件夹路径按 Host 分隔
	parsed, err := url.Parse(Config.BaseURL)
	if err != nil {
		fmt.Printf("解析 baseURL 失败: %v\n", err)
		os.Exit(1)
	}
	host := parsed.Host
	staticPath := "static/" + host

	// ---------- 通信 + Cookie 验证 ----------
	sysInfoURL := Config.BaseURL + "/utils/sysinfo"
	req, _ := http.NewRequest("GET", sysInfoURL, nil)
	req.Header.Set("Cookie", Config.Cookie)

	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("无法连接目标: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("读取响应失败: %v\n", err)
		os.Exit(1)
	}

	if !strings.Contains(string(body), "success") {
		fmt.Println("通信失败：未在响应中检测到 'success'，可能是 Cookie 无效或无权限访问")
		os.Exit(1)
	}

	fmt.Printf("连接 %s 成功，准备开始轮询，每 %d 分钟执行一次...\n", Config.BaseURL, *interval)

	// ---------- 主循环 ----------
	for {
		newIDs := find.FetchNewIDs(Config.BaseURL, staticPath)
		viewdetail.QueryDetails(newIDs, Config.BaseURL, staticPath)

		// 打印轮询完成时间
		fmt.Println("OK+", time.Now().Format("2006-01-02 15:04:05"))
		os.Stdout.Sync()

		time.Sleep(time.Duration(*interval) * time.Minute)
	}
}
