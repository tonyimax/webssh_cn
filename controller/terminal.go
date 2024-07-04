// Package controller : 控制器
package controller

import (
	"fmt"                          //格式化
	"github.com/gin-gonic/gin"     //Gin框架
	"github.com/gorilla/websocket" //websocket库
	"net/http"                     //http库
	"strconv"                      //字符串转换库
	"time"                         //时间日期库
	"webssh/core"                  //本地core库，用于处理SSH与SFTP
)

// 指定将HTTP连接升级为WebSocket连接的参数
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024, //读包缓冲
	WriteBufferSize: 1024, //写包缓冲
	//跨域请求支持
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// TermWs 获取终端websocket
// c: Gin框架上下文
// timeout: 连接超时
// 返回ResponseBody结构
func TermWs(c *gin.Context, timeout time.Duration) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"} //响应成功消息
	defer TimeCost(time.Now(), &responseBody)    //响应超时计算
	sshInfo := c.DefaultQuery("sshInfo", "")     //查询SSH客户端信息
	//查询行与列值,如空，使用默认值
	cols := c.DefaultQuery("cols", "150")
	rows := c.DefaultQuery("rows", "35")
	//连接超时提示
	closeTip := c.DefaultQuery("closeTip", "Connection timed out!")
	//转换为整数
	col, _ := strconv.Atoi(cols)
	row, _ := strconv.Atoi(rows)
	//SSH客户端信息反序列化为SSHClient对象
	sshClient, err := core.DecodedMsgToSSHClient(sshInfo)
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//升级HTTP连接为Websocket连接
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	//升级失败
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//生成SSH客户端
	err = sshClient.GenerateClient()
	//生成出错
	if err != nil {
		wsConn.WriteMessage(1, []byte(err.Error())) //向Websocket客户端发送错误信息
		wsConn.Close()                              //关闭Websocket连接
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	sshClient.InitTerminal(wsConn, row, col)     //初始化终端
	sshClient.Connect(wsConn, timeout, closeTip) //连接Websocket
	return &responseBody
}
