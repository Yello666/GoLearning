# JWT的学习

### 1.引包

首先要看Goland的设置是否启用了gomodule，一定要开不然怎么go get go tidy都无济于事

**打开ide的go设置-go模块-启用go模块集成，这一步非常重要**

然后使用

```go
go mod init
go get github.com/golang-jwt/jwt/v5
go mod tidy
```

项目架构为

goPractice

--main

----main.go

--go.mod

go.mod一定是在根目录下

输入

```go
import "github.com/golang-jwt/jwt/v5"
```

即可以使用jwt

### 2.使用jwt的框架

```go
package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

// 1.定义Claims，其它字段为身份验证所需要的字段
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 2.写生成JWT的函数
func GenerateToken(userID, username string) (string, error) {
	//2.1使用claim结构体来存储用户信息
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "YQ app",
		},
	}
	//2.2使用claim创建并返回token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("YQ FOREVER"))
}
//3.写验证jwt的函数
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{} 
  //3.1解析token
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { 
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte("YQ FOREVER"), nil
	})
	if err != nil {
		return nil, err
  }
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
  //3.2返回解析出来的用户名和id
	return claims, nil
}

// 4.写JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header missing"})
			return
		}

		tokenString := authHeader[7:] // 跳过 "Bearer " 前缀
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token format"})
			return
		}

		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token", "details": err.Error()})
			return
		}

		c.Set("claims", claims) // 将用户信息存入上下文
		c.Next()              // 继续处理请求
	}
}

//5.定义main函数中的登录路由和权限资源路由，并定义中间件
func main() {
	r := gin.Default()

	// 公开路由组
	public := r.Group("/")
	{
		public.POST("/login", loginHandler)
	}

	// 受保护的路由组
	protected := r.Group("/protected")
	protected.Use(AuthMiddleware()) // 应用中间件
	{
		protected.GET("/", protectedHandler)
	}

	r.Run(":8080") // 启动服务器
}

// 6.写登录的处理函数，负责调用生成jwt的函数并返回token
func loginHandler(c *gin.Context) {
	username := c.Query("username")
	userid := c.Query("userid")

	token, err := GenerateToken(userid, username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"token":  token,
		"status": "success",
	})

}

//7.写受限资源的处理函数，要求从中间件中获取用户名，同时可以定义用户权限
func protectedHandler(c *gin.Context) {
	user, exists := c.Get("claims")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}

	claims, ok := user.(*Claims) //获取解析出来的username，userid
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user claims"})
		return
	}
	if claims.Username != "Yegg" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "you are silly egg!"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Hello, %s!", claims.Username)})
}

```



### 3.jwt简介与上述函数详解

#### 3.1JWT的组成部分

一个JWT为
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
由三个部分组成，由两个点分隔开
**HEADER**：
{
  "alg": "HS256", //签名算法
  "typ": "JWT" //令牌类型
}
header会被编码成一个字符串，作为第一部分

**PAYLOAD**：
{
  "user_id": "1",        // 用户 ID
  "exp": 1706347400,     // 过期时间（Unix 时间戳，例：2024年12月1日 00:00:00）
  "username": "user_admin" // 用户名
}
payload会被编码为一个字符串，作为第二部分

**signature**（签名，防止数据被篡改）
一个字符串，将header.payload和密钥使用header中的签名算法HMAC-SHA256编成一个字符串
作为第三部分

一个JWT由以上三部分组成，并以.连起来，这就是传递给JWT验证函数的tokenString

#### 3.2 携带token的json请求示例

```
    Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
    Content-Type: application/json
    Accept: application/json
    {body}
```



### 3.3ValidateToken函数详解

```go
func ValidateToken(tokenString string) (*Claims, error) {
    claims := &Claims{} //使用指针来存储解析后的jwt payload

    //第一个参数是要检验的字符串，第二个参数是存储解析后的jwt payload，
    //第三个参数是一个回调函数，回调函数就是当成参数传递的函数，这个函数在ParseWithClaims会被调用，所以称之为回调函数
    //在ParseWithClaims中会声明一个token *jwt.Token，里面存储header，payload，signature，同时作为回调函数的参数传进去
    token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
       if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok { //检查使用的签名加密方法是否为HMAC
          return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
       }
       return []byte("YQ FOREVER"), nil //回调函数返回密钥，ParseWithClaims会检验signature的密钥和这个是否一样
    })

    if err != nil {
       return nil, err
    }
    //返回的token对象的结构
    /*
       type Token struct {
           Raw       string                 // 原始 JWT 字符串
           Method    SigningMethod          // 签名方法（从 Header 解析）
           Header    map[string]interface{} // JWT 头部
           Claims    Claims                 // JWT 载荷（解析为 Claims 结构体）
           Signature string                 // 签名部分
           Valid     bool                   // 验证结果标志
       }
    */
    if !token.Valid { //如果为false证明密钥不对或者已经过期
       return nil, fmt.Errorf("invalid token")
    }
    return claims, nil
}
```

