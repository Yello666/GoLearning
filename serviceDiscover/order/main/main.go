/*
实现了一个简单的订单查询服务并将其注册到注册中心了
使用：
http://127.0.0.1:8082/orders/user-1 GET
*/
package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 1. 创建Consul客户端
	consulClient, err := createConsulClient()
	if err != nil {
		log.Fatalf("初始化Consul客户端失败: %v", err)
	}

	// 2. 注册服务到Consul
	serviceID := "order-service-1"
	if err := registerService(consulClient, serviceID); err != nil {
		log.Fatalf("注册服务失败: %v", err)
	}
	//注册服务到时候就defer把服务注销
	defer deregisterService(consulClient, serviceID)

	//3. 初始化Gin引擎
	r := gin.Default()

	// 定义路由
	r.GET("/orders/:userID", GetOrdersByUserID)
	r.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 启动服务 使用异步启动，新开一个线程来执行这个函数，主线程执行下面的代码
	//这样写可以进行让HTTP服务在后台运行，不阻塞主线程，允许主线程继续执行信号监听和资源清理逻辑
	//可以实现使用sigint信号优雅退出
	go func() {
		log.Println("订单服务已启动，监听端口8082")
		if err := r.Run(":8082"); err != nil {
			log.Fatalf("启动订单服务失败: %v", err)
		}
	}()

	// 优雅退出
	waitForShutdown()
	log.Println("订单服务已关闭")
}

func createConsulClient() (*api.Client, error) {
	//创建一个与consul服务发现与注册系统通信的客户端
	config := api.DefaultConfig()
	return api.NewClient(config)
}

func registerService(client *api.Client, serviceID string) error {
	//1.声明一个结构体变量（实现接口）
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "order-service",
		Port:    8082,
		Address: "localhost",
		//一个健康检查接口
		//规定了consul向那个路由发get请求来进行健康检查
		Check: &api.AgentServiceCheck{
			HTTP:                           "http://localhost:8082/health", //consul向这个url发get请求，服务器一定要实现这个路由
			Interval:                       "20s",                          //consul每隔10s向health发送一次请求
			Timeout:                        "5s",                           //如果请求在5s内没有想要，视为失败
			DeregisterCriticalServiceAfter: "1m",                           //当服务连续多次健康检查失败时，1min后自动删掉服务
		},
	}

	return client.Agent().ServiceRegister(registration)
}

func deregisterService(client *api.Client, serviceID string) {
	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Printf("注销服务失败: %v", err)
	}
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}

// 订单结构体
type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Product   string    `json:"product"`
	Amount    float64   `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

// 模拟订单数据库
var orders = []Order{
	{ID: "order-1", UserID: "user-1", Product: "手机", Amount: 4999.00, CreatedAt: time.Now()},
	{ID: "order-2", UserID: "user-1", Product: "耳机", Amount: 999.00, CreatedAt: time.Now().Add(-24 * time.Hour)},
	{ID: "order-3", UserID: "user-2", Product: "电脑", Amount: 8999.00, CreatedAt: time.Now().Add(-48 * time.Hour)},
}

// 根据用户ID查询订单
func GetOrdersByUserID(c *gin.Context) {
	userID := c.Param("userID")

	// 过滤订单
	var userOrders []Order
	for _, order := range orders {
		if order.UserID == userID {
			userOrders = append(userOrders, order)
		}
	}

	if len(userOrders) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "未找到该用户的订单"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "订单查询成功",
		"orders":  userOrders,
	})
}
