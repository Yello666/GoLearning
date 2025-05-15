package main

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"time"
)

// Claims 定义JWT中的用户信息
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateToken 生成JWT
func GenerateToken(userID, username string) (string, error) {
	//使用claim结构体来存储用户信息
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			Issuer:    "YQ app",
		},
	}
	//使用claim创建并返回token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("YQ FOREVER"))
}

/*
一个token string为
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
由三个部分组成，由两个点分隔开
HEADER：
{
  "alg": "HS256", //签名算法
  "typ": "JWT" //令牌类型
}
header会被编码成一个字符串，作为第一部分

PAYLOAD：
{
  "user_id": "1",        // 用户 ID
  "exp": 1706347400,     // 过期时间（Unix 时间戳，例：2024年12月1日 00:00:00）
  "username": "user_admin" // 用户名
}
payload会被编码为一个字符串，作为第二部分

signature（签名，防止数据被篡改）
一个字符串，将header.payload和密钥使用header中的签名算法HMAC-SHA256编成一个字符串
作为第三部分

一个JWT由以上三部分组成，并以.连起来，这就是传递进去的tokenString
*/

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
	if !token.Valid { //表示
		return nil, fmt.Errorf("invalid token")
	}
	claims, ok := token.Claims.(*Claims)
	if ok && token.Valid {
		username := claims.Username
		if username != "Yegg" {
			return nil, fmt.Errorf("Access denied")
		}

	}

	return claims, nil
}

// AuthMiddleware JWT认证中间件
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		/*
			携带token的json请求示例：
			//请求头：
			Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c
			Content-Type: application/json
			Accept: application/json
			{body}
		*/
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

		c.Set("user", claims) // 将用户信息存入上下文
		c.Next()              // 继续处理请求
	}
}

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

// 登录接口 - 生成Token
/*
处理逻辑：
1.读取用户名和id
2.生成token
*/
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

// 受保护的接口
func protectedHandler(c *gin.Context) {
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found in context"})
		return
	}

	claims, ok := user.(*Claims)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user claims"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Hello, %s!", claims.Username)})
}
