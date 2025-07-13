package Bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// SendToWeCom 会从 config/botkey.txt 中读取 webhook URL 并发送内容
func SendToWeCom(content string) error {
	webhookURL, err := readWebhookURL()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"msgtype": "text",
		"text": map[string]string{
			"content": content,
		},
	}

	data, _ := json.Marshal(payload)

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("企业微信推送失败")
	}

	return nil
}

func readWebhookURL() (string, error) {
	data, err := ioutil.ReadFile("config/botkey.txt")
	if err != nil {
		return "", err
	}
	url := strings.TrimSpace(string(data))
	if !strings.HasPrefix(url, "https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=") {
		return "", errors.New("config/botkey.txt 中 webhook 格式不正确")
	}
	return url, nil
}
