## Consul服务发现与注册框架在gin框架中的使用

本文主要解决以下问题：

如何在gin框架中使用consul框架实现服务发现与注册



### 第一步 下载consul服务器

macOS

```
brew install consul
service start consul
```

使用

```
consul agent -dev
```

可以启动consul服务器，它是用来健康检查的，会定期给服务发请求来检测服务是否还能用。写完代码再启动consul服务器，现在先不启动。

### 第二步 写好一个或多个服务并进行注册（微服务拆分）

写好你要注册的服务，要求每个服务都可以独立运行

项目框架：

```
.
├── apiGateway
│   ├── go.mod
│   ├── go.sum
│   └── main
│       └── main.go
├── order
│   ├── go.mod
│   ├── go.sum
│   └── main
│       └── main.go
└── user
    ├── go.mod
    ├── go.sum
    └── main
        └── main.go

```

这个例子就是有一个用户服务和一个订单服务，apiGateway负责发现服务和反向代理（请求转发）

#### 用户服务代码：

核心逻辑部分：

```go
/*
实现了一个简单的用户注册服务并将其注册到注册中心了
使用：
http://127.0.0.1:8083/users/register POST

	{
		"user_id":"12345,
		"user_name":"Y"
	}
	可以进行用户注册
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

```

其它函数实现

```go
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

```



##### 解析：

1.引包

```go
import (
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/consul/api" //实现consul框架
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)
```

2.创建consul客户端

```go
func createConsulClient() (*api.Client, error) {
	config := api.DefaultConfig()
	return api.NewClient(config)
}
```

在main函数调用这个函数可以创建并返回一个consul客户端实例，用于与consul服务进行交互（服务注册，发现，健康检查等）

3.注册服务到consul

```go
func registerService(client *api.Client, serviceID string) error {
	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    "user-service",
		Port:    8083,//服务运行端口
		Address: "localhost",//服务ip
		Check: &api.AgentServiceCheck{//健康检查参数
			HTTP:                           "http://localhost:8083/health",//健康检查的请求访问url，这是需要自己实现的
			Interval:                       "20s",//时间间隔
			Timeout:                        "5s",//发请求后过多少秒算超时
			DeregisterCriticalServiceAfter: "1m",//超时多少秒就注销服务
		},
	}
	return client.Agent().ServiceRegister(registration)
}
```

接收一个serviceID和刚刚创建的consulClient，在里面写上服务器的ID和服务名称等参数。在main函数中调用。

ID是唯一标识，注册的服务ID不可以相同，Name是服务名称，尽量也不要相同，因为转发请求的时候是根据服务名称进行转发的。



记得defer一下注销函数。

```go
func deregisterService(client *api.Client, serviceID string) {
	if err := client.Agent().ServiceDeregister(serviceID); err != nil {
		log.Fatalf("注销服务失败")
	}
}
```

4.写自己的服务和启动服务。注意要实现健康检查接口，就是/health接口，返回200就行了

#### 订单服务代码

雷同的，就不再赘述了

```go
/*
实现了一个简单的订单查询服务并将其注册到注册中心了
使用：
http://127.0.0.1:8082/orders/user-1 GET
可以查到用户的user-1的订单，已经提前存储好了数据
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

```

#### API网关代码

```go
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

	// 路由配置，访问这个路由的会被转发到user服务
	r.POST("users/register", func(c *gin.Context) {
		userServiceProxy.ServeHTTP(c.Writer, c.Request)
	})

	r.GET("orders/:userID", func(c *gin.Context) {
		orderServiceProxy.ServeHTTP(c.Writer, c.Request)
	})


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

```

感觉注释说的很清楚了，就不再补充过程了。

注意！

负载均衡那里可以有自己的负载均衡算法，我这只是取第一个实例。

路由匹配可以使用通配符，但一定要和真实服务的有所不一样，不然可能会发生循环重定向。（本人在做项目时深受毒害，debug2小时才得出的结论）。打个比方，你真实服务的地址是（http://xxx:8081/goods),  api网关的反向代理路由设置是，访问（http://xxx:8080/goods)会转发到（http://xxx:8081/goods)。这样不行，容易发生循环重定向。真实服务地址是（http://xxx:8081/goods/add)，就可以成功转发。

#### 运行

先运行consul服务器

另起终端

```
consul agent -dev
```

然后分别运行这三个服务

当用户服务和商品服务的健康检查结果为200的时候，就可以调用接口做测试了，访问api网关的端口8080和注册过的路由，可以访问到对应的服务。类似于反向代理服务器。

好的这篇文章就是这样，希望能帮到你