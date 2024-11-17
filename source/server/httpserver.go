package server

import (
	"fmt"
	"net/http"
	"nyaedge-center/source"
	"nyaedge-center/source/helper"
	"nyaedge-center/source/node"
	"nyaedge-center/source/zaplogger"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func StartServer(host string, port int, logger *zap.Logger, database *mongo.Client) error {
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	r.Use(zaplogger.ZapLogger(logger))

	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello from the server package!",
		})
	})

	api := r.Group("/api")
	{
		nodeapi := api.Group("/node")
		{
			nodeapi.POST("/up", func(c *gin.Context) {
				var requestBody struct {
					NodeID   string `json:"node_id" binding:"required"`
					EndPoint string `json:"endpoint" binding:"required"`
				}

				if err := c.ShouldBindJSON(&requestBody); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Request"})
					return
				}

				oid, err := bson.ObjectIDFromHex(requestBody.NodeID)
				if err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid nodeID"})
					return
				}

				// 从数据库中获取指定的 Node
				nodeinfo, err := source.GetNodebyid(database, source.DatabaseName, source.NodeCollection, oid)
				if err != nil {
					logger.Error("Error getting node: %v", zap.Error(err))
					c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
					return
				}

				_, err = node.PingNode(requestBody.EndPoint, nodeinfo.NodeSecret)
				if err != nil {
					logger.Error("Error up node: %v", zap.Error(err))
					c.JSON(http.StatusBadRequest, gin.H{
						"error": fmt.Sprintf("Error up node: %v", err),
					})
					return
				}

				// 更新 endpoint
				err = source.UpdateNodeInfo(database, source.DatabaseName, source.NodeCollection, oid, requestBody.EndPoint)
				if err != nil {
					logger.Error("Error update nodeinfo: %v", zap.Error(err))
					c.JSON(http.StatusUnauthorized, gin.H{
						"error": fmt.Sprintf("Error update nodeinfo: %v", err),
					})
					return
				}

				// 返回成功响应
				c.JSON(http.StatusOK, gin.H{
					"message": "Node up successfully",
					"node_id": requestBody.NodeID,
				})
			})

			// challenge 路由
			nodeapi.GET("/challenge", func(c *gin.Context) {
				nodeId := c.Query("nodeId")
				if nodeId == "" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "nodeId is required"})
					return
				}

				jwtHelper, err := helper.GetInstance()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing JWT helper"})
					return
				}

				token, err := jwtHelper.IssueToken(map[string]interface{}{
					"nodeId": nodeId,
				}, "node-challenge", 60*5)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error issuing token"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"challenge": token,
				})
			})

			// token 路由
			nodeapi.POST("/token", func(c *gin.Context) {
				var req struct {
					NodeId    string `json:"nodeId"`
					Signature string `json:"signature"`
					Challenge string `json:"challenge"`
				}

				// 处理 JSON 请求体
				contentType := c.Request.Header.Get("Content-Type")
				if strings.HasPrefix(contentType, "application/json") {
					if err := c.ShouldBindJSON(&req); err != nil {
						c.JSON(http.StatusBadRequest, gin.H{"error": "400 badrequest"})
						return
					}
				} else {
					c.JSON(http.StatusBadRequest, gin.H{"error": "400 badrequest"})
					return
				}

				jwtHelper, err := helper.GetInstance()
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error initializing JWT helper"})
					return
				}

				token, err := jwtHelper.VerifyToken(req.Challenge, "node-challenge")
				if err != nil {
					c.JSON(http.StatusForbidden, gin.H{"error": "Invalid challenge token"})
					return
				}

				claims, ok := token.Claims.(jwt.MapClaims)
				if !ok || !token.Valid {
					c.JSON(http.StatusForbidden, gin.H{"error": "Invalid challenge token"})
					return
				}
				nodeIdFromToken, ok := claims["data"].(map[string]interface{})["nodeId"].(string)
				if !ok || nodeIdFromToken != req.NodeId {
					c.JSON(http.StatusForbidden, gin.H{"error": "Node ID mismatch"})
					return
				}

				// 将 nodeId 转换为 ObjectID
				oid, err := bson.ObjectIDFromHex(req.NodeId)
				if err != nil {
					logger.Error("Invalid nodeId: %v", zap.Error(err))
					c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid nodeId"})
					return
				}

				// 从数据库中获取指定的 Node
				node, err := source.GetNodebyid(database, source.DatabaseName, source.NodeCollection, oid)
				if err != nil {
					logger.Error("Error getting node: %v", zap.Error(err))
					c.JSON(http.StatusNotFound, gin.H{"error": "Node not found"})
					return
				}

				if !source.ComputeSignature(req.Challenge, req.Signature, node.NodeSecret) {
					c.JSON(http.StatusForbidden, gin.H{"error": "Invalid signature"})
					return
				}

				newToken, err := jwtHelper.IssueToken(map[string]interface{}{
					"nodeId": req.NodeId,
				}, "node", 60*60*24)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Error issuing token"})
					return
				}

				c.JSON(http.StatusOK, gin.H{
					"token": newToken,
					"ttl":   1000 * 60 * 60 * 24, // 24小时
				})
			})

		}

		adminapi := api.Group("/admin")
		{
			adminapi.GET("/createnode", func(c *gin.Context) {
				name := c.DefaultQuery("name", "")
				if name == "" {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": "400 badrequest",
					})
					return
				}

				nodeInfo, err := source.CreateNode(database, source.DatabaseName, source.NodeCollection, name)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create node"})
					return
				}

				// 返回创建的节点信息
				c.JSON(http.StatusOK, gin.H{
					"name":       nodeInfo.Name,
					"id":         nodeInfo.NodeID,
					"nodeSecret": nodeInfo.NodeSecret,
				})

			})
		}

	}

	address := fmt.Sprintf("%s:%d", host, port)

	logger.Info(fmt.Sprintf("Server is running at http://%s", address))
	return r.Run(address)
}
