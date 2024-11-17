package source

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"nyaedge-center/source/helper"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// computeSignature 计算并验证签名
func ComputeSignature(challenge, signature, secret string) bool {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(challenge))
	expectedSignature := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// verifyClusterRequest 验证请求
func verifyNodeRequest(c *gin.Context) bool {
	// 从请求头中获取 Authorization 字段
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		return false
	}

	// 按空格分割，获取令牌部分
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return false
	}
	tokenString := parts[1]

	// 获取 JWTHelper 实例
	jwtHelper, err := helper.GetInstance()
	if err != nil {
		return false
	}

	// 验证令牌，受众为 'cluster'
	token, err := jwtHelper.VerifyToken(tokenString, "node")
	if err != nil {
		return false
	}

	// 检查令牌是否有效
	if !token.Valid {
		return false
	}

	// 提取声明（claims）并进行额外的验证
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	// 检查特定的声明，如 clusterId
	if _, exists := claims["data"].(map[string]interface{})["nodeId"]; !exists {
		return false
	}

	return true
}
