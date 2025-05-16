## redis in go-gin

### 1.完整代码示例

```go
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

var ctx = context.Background() //用于向redis同步客户端的状态信息，避免客户端断开连接时，redis还在查询
var db *gorm.DB
var redisClient *redis.Client//登录redisclient，用于进行redis中的键值对的增删查改操作

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

```

### 2.步骤解析

#### 2.1 引包

```go
import (
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8" //redis
	"gorm.io/driver/sqlite" //一个文件，可以用来存储数据
	"gorm.io/gorm"
)
```

#### 2.2登录redis

```go
var ctx = context.Background() //用于向redis同步客户端的状态信息，避免客户端断开连接时，redis还在查询
var redisClient *redis.Client//登录redisclient，用于进行redis中的键值对的增删查改操作

func redisInit(){
  redisClient = redis.NewClient(&redis.Options{
		Addr:     "192.168.64.2:6379", //redis所在ip与端口
		Password: "123456", //密码
		DB:       0, //数据库编号（有0-15）
	})

	_, err = redisClient.Ping(ctx).Result() //连接后尝试能否ping通
	if err != nil {
		log.Fatalf("Failed to connect to redis: %v", err)
	}
}
}
```

#### 2.3 redis的增删查改

**增加或修改**键值对，设置缓存过期时间（时间设置为0就是不过期）

```go
var user User //一个结构体
c.ShouldBindJSON(&user); //1.从客户端的json数据那里获取结构体信息
// 2.序列化为 JSON格式（变成json才可以存进redis，一个结构体是存不进去的）
	userJSON, _:= json.Marshal(user)
	//3.设置在redis里面的key
	redisKey := fmt.Sprintf("user:%d", user.ID)
//4.设置key对应的value以及过期时间（一小时）
	err = redisClient.Set(ctx, redisKey, userJSON, time.Hour).Err()
	if err != nil {
		log.Printf("set redis cache failed:%v", err)
	}
```

**删除键值对**

```go
	var user User
//通过数据库查找来获取user的值
	db.First(&user, id)
	// 获得userID，获得了redisKey，就可以删除Redis缓存
	redisKey := fmt.Sprintf("user:%d", id)
	if err := redisClient.Del(ctx, redisKey).Err(); err != nil {
		log.Printf("Failed to delete redis cache: %v", err)
	}

```

**查看键值对**

```go
idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	redisKey := fmt.Sprintf("user:%d", id)
	//1.尝试从redis缓存读取
	userJSON, err := redisClient.Get(ctx, redisKey).Result()
	if err == nil {
		//缓存命中,读取redis中的json值，并反序列化为结构体
		var user User
		if err := json.Unmarshal([]byte(userJSON), &user); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "failed to parse user data",
			})
			return
		}
	return user
	}
```

