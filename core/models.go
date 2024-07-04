package core // Package core 核心包
//导入依赖包
import (
	"github.com/gorilla/websocket" //websocket包
	"github.com/pkg/sftp"          //sftp包
	"golang.org/x/crypto/ssh"      //ssh包
	"io"                           //io操作
	"log"                          //日志记录
	"unicode/utf8"                 //utf8字符编码解码
)

// WcList Websocket连接池
var WcList []*WriteCounter

// WriteCounter 结构体
type WriteCounter struct {
	Total int    //总数
	Id    string //标识
}

// 为WriteCounter结构实现Write方法
// p:要写入的字节数组
// 返回成功写入的字节数与错误码
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)   //取字节长度
	wc.Total += n //统计写入字节数
	return n, nil //返回成功写入的字节数与错误码
}

// 输出Websocket连接对象
type wsOutput struct {
	ws *websocket.Conn
}

// Write: 为wsOutput实现Write方法
// p:要写入的字节数组
func (w *wsOutput) Write(p []byte) (int, error) {
	// 处理非utf8字符
	if !utf8.Valid(p) {
		bufStr := string(p) //转换为字符串
		//rune表示int32,[]rune相当于[]int32
		buf := make([]rune, 0, len(bufStr)) //分配内存用于存储上面bufStr字符串
		//遍历字符串
		for _, r := range bufStr {
			//如果是非法unicode字符
			if r == utf8.RuneError {
				buf = append(buf, []rune("@")...) //替换为@
			} else {
				buf = append(buf, r) //拼接字符
			}
		}
		p = []byte(string(buf)) //字符串强制转换为字节数组
	}
	//向websocket发送文本消息
	err := w.ws.WriteMessage(websocket.TextMessage, p)
	//返回已发送的字节长度与错误码
	return len(p), err
}

// SSHClient 结构体
type SSHClient struct {
	Username  string         `json:"username"`  //用户名
	Password  string         `json:"password"`  //密码
	IPAddress string         `json:"ipaddress"` //IP地址
	Port      int            `json:"port"`      //端口
	LoginType int            `json:"logintype"` //登陆类型
	Client    *ssh.Client    //SSH客户端
	Sftp      *sftp.Client   //SFTP客户端
	StdinPipe io.WriteCloser //写IO接口，这里表示标准输入管道
	Session   *ssh.Session   //SSH会话
}

// NewSSHClient 创建新的SSH客户端实例并使用默认用户名root及默认端口22
func NewSSHClient() SSHClient {
	client := SSHClient{}    //SSH客户端实例
	client.Username = "root" //默认用户名
	client.Port = 22         //默认端口
	return client            //返回SSH客户端实例
}

// Close 关闭SSHClient结构中StdinPipe, Session, Sftp, Client
// 这4个字符中所有打开的连接
func (sclient *SSHClient) Close() {
	//Close方法所有语句执行完成后，执行这个函数
	defer func() {
		if err := recover(); err != nil {
			log.Println("SSHClient Close recover from panic: ", err)
		}
	}()
	//关闭标准输入通道
	if sclient.StdinPipe != nil {
		sclient.StdinPipe.Close()
		sclient.StdinPipe = nil
	}
	//关闭会话
	if sclient.Session != nil {
		sclient.Session.Close()
		sclient.Session = nil
	}
	//关闭SFTP文件传入连接
	if sclient.Sftp != nil {
		sclient.Sftp.Close()
		sclient.Sftp = nil
	}
	//关闭SSH客户端
	if sclient.Client != nil {
		sclient.Client.Close()
		sclient.Client = nil
	}
}
