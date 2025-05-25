/*
实现了一个简单的用户注册服务并将其注册到注册中心了
使用：
http://127.0.0.1:8083/users/register POST

	{
		"user_id":"12345,
		"user_name":"Y"
	}
*/
package main

import (
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var users []User

func main() {
	//1.创建consul客户端
	consulClient, err := createConsulClient()
	if err != nil {
		log.Fatalf("创建consul客户端失败：%v", err)
	}
	//2.注册服务到consul
	serviceID := "user-service-1"
	err = registerService(consulClient, serviceID)
	if err != nil {
		log.Fatalf("注册服务失败:%v", err)
	}
	defer deregisterService(consulClient, serviceID)

	//写服务
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.Status(200)
	})
	r.POST("/users/register", CreateUser)

	//启动服务
	go func() {
		log.Printf("用户注册服务已经启动在端口8083\n")
		if err := r.Run(":8083"); err != nil {
			log.Fatalf("启动用户注册服务失败: %v", err)
		}
	}()
	// 优雅退出
	waitForShutdown()
	log.Println("订单服务已关闭")
}
func createConsulClient() (*api.Client, error) {
	config := api.DefaultConfig()
	return api.NewClient(config)
}

func registerService(client *api.Client, serviceID string) error {
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "user-service",
		Port:    8083,
		Address: "localhost",
		Check: &api.AgentServiceCheck{
			HTTP:                           "http://localhost:8083/health",
			Interval:                       "20s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "1m",
		},
	}
	return client.Agent().ServiceRegister(registration)
}
func deregisterService(client *api.Client, serviceID string) {
	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Fatalf("注销服务失败")
	}
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

type User struct {
	UserID    string    `json:"user_id"`
	UserName  string    `json:"user_name"`
	CreatedAt time.Time `json:"created_at"`
}

// 创建用户

func CreateUser(c *gin.Context) {
	var user User
	err := c.BindJSON(&user)
	if err != nil {
		c.JSON(500, gin.H{
			"status":  "failed",
			"message": err.Error(), //响应直接返回err会输出一个结构体而不是错误信息
		})
		return
	}
	user.CreatedAt = time.Now()
	users = append(users, user)
	c.JSON(200, gin.H{
		"status":  200,
		"message": "success",
	})

}
