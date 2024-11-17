package source

import (
	"context"
	"math/rand"
	"nyaedge-center/source/zaplogger"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

// Node 表示 nodes 集合中的文档结构
type Node struct {
	NodeID       bson.ObjectID `bson:"_id"`
	NodeSecret   string        `bson:"node_secret"`
	Name         string        `bson:"name"`
	NodeEndPoint string        `bson:"node_endpoint"`
	CreateAt     bson.DateTime `bson:"createAt"`
}

var DatabaseName = "nyaedge"
var NodeCollection = "nodes"

// 生成一个随机 SECRET
func GenerateRandomSecret() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	secret := make([]byte, 32)
	for i := range secret {
		secret[i] = charset[rng.Intn(len(charset))]
	}
	return string(secret)
}

// SetupDatabase 连接到 MongoDB
func SetupDatabase(uri string) (*mongo.Client, error) {
	logger := zaplogger.Getlogger()

	clientOptions := options.Client().ApplyURI(uri)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, err
	}

	// 检查连接
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil, err
	}

	logger.Info("Connected to MongoDB")

	return client, nil
}

// EnsureCollection 确保指定的集合存在
func EnsureCollection(client *mongo.Client, dbName, collectionName string) error {
	logger := zaplogger.Getlogger()

	collectionNames, err := client.Database(dbName).ListCollectionNames(context.TODO(), bson.M{})
	if err != nil {
		return err
	}

	// 检查集合是否存在
	collectionExists := false
	for _, name := range collectionNames {
		if name == collectionName {
			collectionExists = true
			break
		}
	}

	// 如果集合不存在，则创建集合
	if !collectionExists {
		err := client.Database(dbName).CreateCollection(context.TODO(), collectionName)
		if err != nil {
			return err
		}
		logger.Debug("Collection created successfully", zap.String("collectionName", collectionName))
	} else {
		logger.Debug("Collection already exists. Skip", zap.String("collectionName", collectionName))
	}

	return nil
}

// 创建节点
func CreateNode(client *mongo.Client, dbName, collectionName, name string) (*Node, error) {
	node := Node{
		NodeID:       bson.NewObjectID(),
		NodeSecret:   GenerateRandomSecret(),
		Name:         name,
		NodeEndPoint: "",
		CreateAt:     bson.DateTime(time.Now().UnixNano() / int64(time.Millisecond)),
	}

	collection := client.Database(dbName).Collection(collectionName)
	_, err := collection.InsertOne(context.TODO(), node)
	if err != nil {
		return nil, err
	}

	return &Node{
		Name:       name,
		NodeID:     node.NodeID,
		NodeSecret: node.NodeSecret,
	}, nil
}

// 更新节点信息
func UpdateNodeInfo(client *mongo.Client, dbName, collectionName string, nodeID bson.ObjectID, endpoint string) error {
	collection := client.Database(dbName).Collection(collectionName)

	// 更新节点的 endpoint
	filter := bson.M{"_id": nodeID}
	update := bson.M{"$set": bson.M{"node_endpoint": endpoint}}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	return err
}

// 获取节点的 secret
func GetNodebyid(client *mongo.Client, dbName, collectionName string, nodeID bson.ObjectID) (*Node, error) {
	collection := client.Database(dbName).Collection(collectionName)

	var node Node
	err := collection.FindOne(context.TODO(), bson.M{"_id": nodeID}).Decode(&node)
	if err != nil {
		return nil, err
	}

	return &node, nil
}
