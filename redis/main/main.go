package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	ID   uint   `json:"id" gorm:"primaryKey"`
	Name string `json:"name" `
	Age  int    `json:"age"`
}

var ctx = context.Background() //？
var db *gorm.DB
var redisClient *redis.Client

func init() {
	var err error
	//sqlite是一个文件数据库，就是一个文件，适合轻量级开发，会在项目目录下创建一个test.db文件
	db, err = gorm.Open(sqlite.Open("test.db"), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	err = db.AutoMigrate(&User{})
	if err != nil {
		return
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     "192.168.64.2:6379",
		Password: "123456",
		DB:       0,
	})

	_, err = redisClient.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
}

func main() {
	r := gin.Default()

	userGroup := r.Group("/user")
	{
		userGroup.GET("/:id", GetUser)
		userGroup.POST("", CreateUser)
		userGroup.PUT("/:id", UpdateUser)
		userGroup.DELETE("/:id", DeleteUser)

	}
	err := r.Run(":8080")
	if err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}

}

func GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid user id",
		})
		return
	}
	redisKey := fmt.Sprintf("user:%d", id)

	//1.尝试从redis缓存读取!!
	userJSON, err := redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		//缓存命中
		var user User
		if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to parse user data",
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status": "success",
			"user":   user,
		})
		return
	}
	//缓存未命中，从数据库中读取
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "user not found",
		})
		return
	}
	//3.将数据写入Redis缓存，设置过期时间为一个小时！！
	userJSONBytes, err := json.Marshal(user)
	if err != nil {
		log.Printf("failed to set redis cache:%v", err)
		//缓存失败也要返回user
	}
	if err := redisClient.Set(ctx, redisKey, userJSONBytes, time.Hour).Err(); err != nil {
		log.Fatalf("failed to set redis cache:%v", err)
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"user":   user,
	})

}

// CreateUser 创建用户
func CreateUser(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 保存到数据库
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}
	// 手动序列化为 JSON
	userJSON, err := json.Marshal(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize user"})
		return
	}
	//保存进缓存里面
	redisKey := fmt.Sprintf("user:%d", user.ID)
	err = redisClient.Set(ctx, redisKey, userJSON, time.Hour).Err()
	if err != nil {
		log.Printf("set redis cache failed:%v", err)
	}
	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"user":   user,
	})
}

type UserUpdate struct {
	Name *string `json:"name"`
	Age  *int    `json:"age"`
}

func UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}
	// 检查用户是否存在
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	//绑定更新数据
	var updateData UserUpdate
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	// 更新数据库
	if err := db.Model(&user).Updates(updateData).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	//删除redis缓存，下次读取时重新下载
	redisKey := fmt.Sprintf("user:%d", id)
	if err := redisClient.Del(ctx, redisKey).Err(); err != nil {
		log.Printf("Failed to delete redis cache: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"user":   user,
	})
}

// DeleteUser 删除用户
func DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// 检查用户是否存在
	var user User
	if err := db.First(&user, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// 删除用户
	if err := db.Delete(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	// 删除Redis缓存
	redisKey := fmt.Sprintf("user:%d", id)
	if err := redisClient.Del(redisClient.Context(), redisKey).Err(); err != nil {
		log.Printf("Failed to delete redis cache: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
