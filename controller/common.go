// Package controller ：控制器
package controller

import (
	"fmt"                      //格式化
	"github.com/gin-gonic/gin" //gin框架
	"time"                     //时间日期库
	"webssh/core"              //本地core库操作SSH及SFTP用
)

// ResponseBody 响应信息结构体
type ResponseBody struct {
	Duration string      //时长
	Data     interface{} //数据
	Msg      string      //消息
}

// TimeCost 响应耗时计算
func TimeCost(start time.Time, body *ResponseBody) {
	body.Duration = time.Since(start).String()
}

// CheckSSH 检查ssh连接是否能连接
func CheckSSH(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}          //初始化响应成功消息体
	defer TimeCost(time.Now(), &responseBody)             //响应时长计算
	sshInfo := c.DefaultQuery("sshInfo", "")              //查询SSH信息
	sshClient, err := core.DecodedMsgToSSHClient(sshInfo) //解决SSH信息为SSH客户端
	//响应出错,替换错误信息
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//生成SSH客户端
	err = sshClient.GenerateClient()
	defer sshClient.Close() //关闭SSH客户端
	//响应出错,替换错误信息
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
	}
	//返回响应信息
	return &responseBody
}
