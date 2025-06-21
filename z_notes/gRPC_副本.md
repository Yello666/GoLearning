## gRPC

##### 简介

gRPC是一个远程过程调用框架，全称是google remote procudure call,作用是可以让一个客户端调用服务器的某个接口，就像调用本地的函数一样，并且支持不同语言，比如，客户端是JAVA，服务器1是python，服务器2是GO都可以调用。

## 1.使用方式

在go的客户端调用go服务器的接口

### 1.1 下载protobuf库 

brew安装不需要配置环境变量，win和linux也许需要

```
brew install protobuf
protoc --version
```

### 1.2 安装对应语言的工具包

终端输入

```go
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

go get安装的是项目依赖，这个是所有项目都适用的，所以使用go install作为一个本地的包 

### 1.3在golang获取项目依赖 

```go
go get google.golang.org/grpc
go mod tidy
```

### 1.4 编写proto文件

#### 1.4.1 创建目录

在项目根目录下创建RPC文件夹，里面创建一个server.proto文件（只要后缀是proto即可）

项目目录：

*-goPractice    -gRPC-server-main-main.go //server*

​				   	*-RPC -service.proto //rpc协议规定*

​				  *-client-main-main.go //client*

​					  *-RPC -service.proto //rpc协议规定*

#### 1.4.2 .proto文件的说明：

```protobuf
syntax = "proto3";//使用语法：

package RPC; //这个文件在项目中所属的包

option go_package = ".;gen_server"; //使用grpc生成的代码放在哪
//.代表在当前目录生成 ;是间隔 gen_server是生成代码的包名 就是在./gen_server下放生成的server代码

service MyService { 
  rpc SayHello (HelloRequest) returns (HelloResponse) {}
}

message HelloRequest {
  string name = 1;
}

message HelloResponse {
  string rsp_message = 1;
}
```

1. service是服务定义关键字 定义了一个叫MyService的服务，里面有一个函数

2. 函数声明以rpc开头，函数名为SayHellow,传入的参数为HelloRequest,传出的参数为HelloResponse。

3. message关键字就是上面的函数的入参和出参的定义

```
message HelloRequest {
  string name = 1;
  // int age=2;
}
```

就是定义了 request，包括一个name，=1的意思是这个字段的编号为1，不是赋值，同一个message里这个编号不能够相同，因为进行二进制编码和解码的时候变量的位置就是由这个编号决定的

#### 1.4.3进入RPC目录，运行proto文件，生成一个二进制解码器 

​	注意指令中要填写本地proto文件的名字

​	注意每种语言生成proto文件的语法都不一样，下面是go语言的

```
protoc --go_out=. --go-grpc_out=. server.proto
```

-  为什么要编译运行.proto文件？生成的东西有什么用？

​	因为网络数据传输最终都要转化成二进制进行传输，平时使用json格式传输，底层也是将json格式转换成二进制格式传输的，这一步就是在干这个结构体格式与二进制的转换工作。

​	当有入参传入我们的接口的时候，传入的是二进制数据，这个生成的代码文件可以将其变成结构体数据，就是给name赋值。剩下的交给服务器处理，服务器返回响应的时候，将rsp_message转换成二进制文件，再发到别的地方。

- 生成了什么代码文件？什么指令生成什么文件？对应的文件实现了什么？

  protoc --go_out=. 生成server.pb.go文件，用来进行二进制数据与结构体数据之间的转换

  protoc --go-grpc_out=. server.proto 生成server_grpc.pb.go文件，用来进行接口的定义与注册。就是生成了以下东西

  - `MyServiceServer` 接口（你需要实现）

  - `MyServiceClient` 结构体（直接使用）

    需要在代码中服务器代码中引入xxx来使用my service client结构体，并实现接口

#### 1.4.4 将RPC文件夹复制到client处，使得client也可以进行数据流的转换

#### 1.4.5 proto文件的语法

定义服务入参与出参

```protobuf
service Greeter {
  // 简单RPC
  rpc SayHello (HelloRequest) returns (HelloReply) {}
  
  // 服务端流式RPC
  rpc LotsOfReplies (HelloRequest) returns (stream HelloReply) {}
  
  // 客户端流式RPC
  rpc LotsOfGreetings (stream HelloRequest) returns (HelloReply) {}
  
  // 双向流式RPC
  rpc BidiHello (stream HelloRequest) returns (stream HelloReply) {}
}
```



定义字段类型

```protobuf
message Person {
  // 字段规则 类型 字段名 = 字段编号;
  string name = 1;
  int32 id = 2;  
  string email = 3;
  
  // 嵌套消息(对应嵌套结构体)
  message Address {
    string street = 1;
    string city = 2;
  }
  Address address = 4;
  
  // 枚举
  enum PhoneType {
    MOBILE = 0;  // proto3要求第一个枚举值必须为0
    HOME = 1;
    WORK = 2;
  }
  
  message PhoneNumber {
    string number = 1;
    PhoneType type = 2;
  }
  
  repeated PhoneNumber phones = 5;  // 重复字段（在转换成结构体的时候会被映射为切片类型）
  
  map<string, string> properties = 6;  // 映射类型
}
```

##### 字段编号规则：

- 1-15：占用1个字节（适合频繁使用的字段）
- 16-2047：占用2个字节
- 不可重复使用，已删除的编号也不应重用

##### 数据类型

| .proto类型 | Go类型  | 说明                     |
| ---------- | ------- | ------------------------ |
| double     | float64 | 双精度浮点               |
| float      | float32 | 单精度浮点               |
| int32      | int32   | 变长编码，适合负数       |
| int64      | int64   | 变长编码                 |
| uint32     | uint32  | 变长编码                 |
| uint64     | uint64  | 变长编码                 |
| sint32     | int32   | 适合负数的变长编码       |
| sint64     | int64   | 适合负数的变长编码       |
| fixed32    | uint32  | 固定4字节，适合大值      |
| fixed64    | uint64  | 固定8字节，适合大值      |
| sfixed32   | int32   | 固定4字节                |
| sfixed64   | int64   | 固定8字节                |
| bool       | bool    | 布尔值                   |
| string     | string  | UTF-8或7-bit ASCII字符串 |
| bytes      | []byte  | 任意字节序列             |

### 1.5 编写server main.go

go.mod 里面加上自己的模块名称

```
module github.com/Yello666/GoLearning/goPrctice/gRPC
```

引入gRPC目录下的任意一个包

Server 

Main.go:

```go
package main

import (
	"context"
	"fmt"
	pb "github.com/Yello666/GoLearning/goPrctice/gRPC/server/RPC"
	"google.golang.org/grpc"
	"net"
)
//实现接口
type server struct {
	pb.UnimplementedMyServiceServer 
}

func (s *server) SayHello(ctx context.Context, req *pb.HelloRequest) (*pb.HelloResponse, error) {
	return &pb.HelloResponse{RspMessage: "hello" + req.Name}, nil
}

func main() {
	listen, _ := net.Listen("tcp", ":8080")
	grpcServer := grpc.NewServer()
	pb.RegisterMyServiceServer(grpcServer, &server{})
	err := grpcServer.Serve(listen)
	if err != nil {
		fmt.Println("error in sevice running:%v", err)
		return
	}
}

```

编写客户端，差不多的

```go
package main

import (
	"context"
	"fmt"
	pb "github.com/Yello666/GoLearning/goPrctice/gRPC/server/RPC"
	"google.golang.org/grpc"
)

func main() {
	conn, _ := grpc.Dial("localhost:8080", grpc.WithInsecure())
	client := pb.NewMyServiceClient(conn)                                            // 使用 _grpc.pb.go 生成的客户端
	res, _ := client.SayHello(context.Background(), &pb.HelloRequest{Name: "Hello"}) // 使用 service.pb.go 生成的 Request
	fmt.Println(res.RspMessage)                                                      // 使用 service.pb.go 生成的方法
}

```

