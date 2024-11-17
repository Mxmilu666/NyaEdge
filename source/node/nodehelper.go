package node

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"nyaedge-center/source"
)

// createHash 使用密钥加盐生成哈希值
func createHash(nonce, secretKey, timestamp string) string {
	saltedNonce := nonce + secretKey + timestamp
	hash := sha256.New()
	hash.Write([]byte(saltedNonce))
	return hex.EncodeToString(hash.Sum(nil))
}

// PingNode 获取节点是否在线
func PingNode(endpoint, secretKey string) (string, error) {
	nonce := source.GenerateRandomSecret()
	timestamp := time.Now().UTC().Format(time.RFC3339)
	hashValue := createHash(nonce, secretKey, timestamp)

	url := endpoint + "/api/node/ping"
	payload := strings.NewReader(fmt.Sprintf(`{"nonce":"%s","timestamp":"%s","hash":"%s"}`, nonce, timestamp, hashValue))
	req, err := http.NewRequest("GET", url, payload)
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}
