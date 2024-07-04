// Package core ：核心包
package core

import (
	"encoding/base64"              //base64编码
	"encoding/json"                //json编码
	"fmt"                          //格式化输入输出
	"github.com/gorilla/websocket" //websocket库
	"golang.org/x/crypto/ssh"      //ssh库
	"log"                          //日志库
	"net"                          //网络库
	"strconv"                      //字符串转换库
	"strings"                      //字符串库
	"time"                         //日期时间库
)

// DecodedMsgToSSHClient 解码字符串为SSH客户端信息
// 返回SSH客户端与错误码
func DecodedMsgToSSHClient(sshInfo string) (SSHClient, error) {
	client := NewSSHClient()                                 //SSH客户端实例
	decoded, err := base64.StdEncoding.DecodeString(sshInfo) //base64编码
	if err != nil {
		return client, err
	}
	err = json.Unmarshal(decoded, &client) //JSON反序列化
	if err != nil {
		return client, err
	}
	//如果IPAddress字段含有:与首字符不是[
	if strings.Contains(client.IPAddress, ":") && string(client.IPAddress[0]) != "[" {
		client.IPAddress = "[" + client.IPAddress + "]" //为字符串前后添加[]
	}
	return client, nil //返回SSH客户端实例与错误码
}

// GenerateClient 创建ssh客户端
func (sclient *SSHClient) GenerateClient() error {
	//局部变量声明
	var (
		auth         []ssh.AuthMethod  //SSH授权方法
		addr         string            //地址
		clientConfig *ssh.ClientConfig //SSH客户端配置
		client       *ssh.Client       //SSH客户端
		config       ssh.Config        //SSH配置
		err          error             //错误
	)
	auth = make([]ssh.AuthMethod, 0) //分配授权方法内存
	//使用密码登陆
	if sclient.LoginType == 0 {
		auth = append(auth, ssh.Password(sclient.Password)) //添加SSH密码
	} else { //使用密钥登陆
		//取私有密钥签名
		if signer, err := ssh.ParsePrivateKey([]byte(sclient.Password)); err != nil {
			return err
		} else {
			auth = append(auth, ssh.PublicKeys(signer)) //使用SSH公钥
		}
	}
	//SSH配置
	config = ssh.Config{
		//SSH加密类型
		Ciphers: []string{"aes128-ctr", "aes192-ctr", "aes256-ctr", "aes128-gcm@openssh.com", "arcfour256", "arcfour128", "aes128-cbc", "3des-cbc", "aes192-cbc", "aes256-cbc"},
	}
	//SSH客户端配置
	clientConfig = &ssh.ClientConfig{
		User:    sclient.Username, //用户名
		Auth:    auth,             //授权
		Timeout: 5 * time.Second,  //超时
		Config:  config,           //配置
		//回调
		HostKeyCallback: func(hostname string, //主机名
			remote net.Addr, //地址
			key ssh.PublicKey) error {
			return nil
		},
	}
	//格式化地址为 IP地址:端口 形式
	addr = fmt.Sprintf("%s:%d", sclient.IPAddress, sclient.Port)
	//连接到SSH服务器
	//连接类型为tcp
	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return err
	}
	sclient.Client = client //存储连接成功的SSH客户端实例到SSHClient结构的Client字段
	return nil
}

// InitTerminal 初始化终端
// ws : WebSocket连接对象
// rows : 行数
// cols : 列数
func (sclient *SSHClient) InitTerminal(ws *websocket.Conn, rows, cols int) *SSHClient {
	sshSession, err := sclient.Client.NewSession() //创建SSH会话
	if err != nil {
		log.Println(err)
		return nil
	}
	sclient.Session = sshSession                  //保存SSH会话
	sclient.StdinPipe, _ = sshSession.StdinPipe() //保存标准输入管道
	wsOutput := new(wsOutput)                     //实例化Websocket对象
	sshSession.Stdout = wsOutput                  //SSH会话标准输出流
	sshSession.Stderr = wsOutput                  //SSH会话标准错误流
	wsOutput.ws = ws                              //保存Websocket实例
	//终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	//请求pty与远程主机上的会话的关联
	if err := sshSession.RequestPty("xterm", rows, cols, modes); err != nil {
		return nil
	}
	//在远程主机上启动一个登录Shell
	if err := sshSession.Shell(); err != nil {
		return nil
	}
	return sclient //返回SSH客户端实例
}

// Connect 连接WebSocket服务端
func (sclient *SSHClient) Connect(ws *websocket.Conn, timeout time.Duration, closeTip string) {
	stopCh := make(chan struct{}) //创建一个传入结构的信道
	//协程处理用户输入
	go func() {
		for {
			// p为用户输入
			_, p, err := ws.ReadMessage() //读取WebSocket消息
			if err != nil {
				close(stopCh) //读取失败，关闭信道
				return
			}
			//忽略ping
			if string(p) == "ping" {
				continue
			}
			//resize消息
			if strings.Contains(string(p), "resize") {
				resizeSlice := strings.Split(string(p), ":")    //分割消息
				rows, _ := strconv.Atoi(resizeSlice[1])         //行
				cols, _ := strconv.Atoi(resizeSlice[2])         //列
				err := sclient.Session.WindowChange(rows, cols) //重置窗口大小
				if err != nil {
					log.Println(err)
					close(stopCh) //重置窗口大小失败，关闭信道
					return
				}
				continue //继续处理下一消息
			}
			//普通消息直接写入输出流
			_, err = sclient.StdinPipe.Write(p)
			if err != nil {
				close(stopCh) //写入失败，关闭信道
				return
			}
		}
	}()

	//延迟执行函数
	defer func() {
		ws.Close()      //关闭Websocket连接(ws表示SSH服务端的Websocket)
		sclient.Close() //关闭SSH客户端
		//recover捕获到panic的错误信息，并允许程序恢复正常的执行流程
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()
	// Websocket超时定时器
	stopTimer := time.NewTimer(timeout)
	defer stopTimer.Stop() //停止定时器

	// 主循环
	for {
		select {
		case <-stopCh: //信息stopCh信道
			return
		case <-stopTimer.C: //处理定时器信道
			ws.WriteMessage(1, []byte(fmt.Sprintf("\u001B[33m%s\u001B[0m", closeTip)))
			return
		}
	}
}
