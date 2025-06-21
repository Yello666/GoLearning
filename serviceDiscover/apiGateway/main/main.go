/*
实现了api网关，相当于一个微服务的nginx，自动发现可以用的微服务示例，并将请求路由到微服务上
还可以实现负载均衡的功能
*/
package main

import (
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api"
)

func main() {
	// 创建Consul客户端
	consulClient, err := createConsulClient()
	if err != nil {
		log.Fatalf("初始化Consul客户端失败: %v", err)
	}

	// 初始化Gin引擎
	r := gin.Default()

	// 创建反向代理
	//这里的serviceName要和当时注册服务时的一样，返回一个RPS，可以将请求转发到对应的服务器上
	userServiceProxy := createServiceProxy(consulClient, "user-service")
	orderServiceProxy := createServiceProxy(consulClient, "order-service")

	// 路由配置
	r.POST("users/register", func(c *gin.Context) {
		userServiceProxy.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("orders/:userID", func(c *gin.Context) {
		orderServiceProxy.ServeHTTP(c.Writer, c.Request)
	})
	//// 在API网关中添加错误处理路由，可以返回一个错误页面的html
	//r.GET("/service-unavailable", func(c *gin.Context) {
	//	c.JSON(http.StatusServiceUnavailable, gin.H{
	//		"code":    503,
	//		"message": "Service temporarily unavailable",
	//		"details": c.Request.Header.Get("X-Error"), // 可选：显示内部错误详情
	//	})
	//})

	// 启动服务
	go func() {
		log.Println("API网关已启动，监听端口8080")
		if err := r.Run(":8080"); err != nil {
			log.Fatalf("启动API网关失败: %v", err)
		}
	}()

	// 优雅退出
	waitForShutdown()
	log.Println("API网关已关闭")
}

func createConsulClient() (*api.Client, error) {
	config := api.DefaultConfig()
	return api.NewClient(config)
}

func createServiceProxy(client *api.Client, serviceName string) *httputil.ReverseProxy {
	//获取已经注册过的可用的服务实例
	director := func(req *http.Request) {
		// 从Consul获取健康的服务实例，叫serverName的服务实例可以有很多个，返回到一个切片里面
		serviceInstances, _, err := client.Health().Service(serviceName, "", true, nil)
		//参数解析：serviceName：从consul获取名为serviveName且健康的服务器，true表示只返回健康的实例
		if err != nil || len(serviceInstances) == 0 {
			// 设置一个非法地址，让 Transport 失败，从而触发 ErrorHandler
			req.URL = &url.URL{}
			return
		}

		// 简单的负载均衡：总是选择第一个实例
		instance := serviceInstances[0] // 实际生产中应该实现更复杂的负载均衡算法
		//修改并添加请求头信息
		target := url.URL{
			Scheme: "http",
			//修改成目标服务器的地址//不可以使用string（）来转换成字符串类型，它会将数字按照ascii码变成字符串
			//而不是“8083”
			Host: instance.Service.Address + ":" + strconv.Itoa(instance.Service.Port),
		}

		req.URL.Scheme = target.Scheme
		req.URL.Host = target.Host
		req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
		req.Host = target.Host
	}
	// 添加错误处理函数
	errorHandler := func(w http.ResponseWriter, r *http.Request, err error) {
		log.Printf("代理请求失败: %v", err)
		http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
	}

	return &httputil.ReverseProxy{
		Director:     director,
		ErrorHandler: errorHandler,
	}
}

func waitForShutdown() {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
}
