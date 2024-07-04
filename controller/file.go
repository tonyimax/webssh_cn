// Package controller : 控制器
package controller

import (
	"fmt"         //格式化输入输出
	"io"          //IO操作
	"net/http"    //HTTP库
	"sort"        //排序库
	"strconv"     //字符串转换
	"strings"     //字符串库
	"time"        //时间日期库
	"webssh/core" //本地core库，用于操作SSH与SFTP

	"github.com/gin-gonic/gin" //Gin框架
)

// File 文件信息结构体
type File struct {
	Name       string //名称
	Size       string //大小
	ModifyTime string //修改日期
	IsDir      bool   //是否为文件夹
}

// 文件大小单位常量
const (
	//1B字节
	BYTE = 1 << (10 * iota)
	//1K千字节
	KILOBYTE
	//1M兆字节
	MEGABYTE
	//1G吉字节
	GIGABYTE
	//1T太字节
	TERABYTE
	//1P 拍字节
	PETABYTE
	//1E艾字节
	EXABYTE
)

// 字节格式化输出
// Bytefmt returns a human-readable byte string of the form 10M, 12.5K, and so forth.  The following units are available:
//
//	E: Exabyte
//	P: Petabyte
//	T: Terabyte
//	G: Gigabyte
//	M: Megabyte
//	K: Kilobyte
//	B: Byte
//
// The unit that results in the smallest number greater than or equal to 1 is always chosen.
func Bytefmt(bytes uint64) string {
	unit := "" //单位
	value := float64(bytes)
	//单位判断
	switch {
	case bytes >= EXABYTE:
		unit = "E"
		value = value / EXABYTE
	case bytes >= PETABYTE:
		unit = "P"
		value = value / PETABYTE
	case bytes >= TERABYTE:
		unit = "T"
		value = value / TERABYTE
	case bytes >= GIGABYTE:
		unit = "G"
		value = value / GIGABYTE
	case bytes >= MEGABYTE:
		unit = "M"
		value = value / MEGABYTE
	case bytes >= KILOBYTE:
		unit = "K"
		value = value / KILOBYTE
	case bytes >= BYTE:
		unit = "B"
	case bytes == 0:
		return "0B"
	}
	//格式化64位浮点数
	result := strconv.FormatFloat(value, 'f', 2, 64)
	//只保留2位小数
	result = strings.TrimSuffix(result, ".00")
	return result + unit //返回格式化后的字符串
}

// 文件数组
type fileSplice []File

// Len 文件数组大小
func (f fileSplice) Len() int { return len(f) }

// Swap 位置交换
func (f fileSplice) Swap(i, j int) { f[i], f[j] = f[j], f[i] }

// Less 比大小
func (f fileSplice) Less(i, j int) bool { return f[i].IsDir }

// UploadFile 上传文件
func UploadFile(c *gin.Context) *ResponseBody {
	//局部变量
	var (
		sshClient core.SSHClient //SSH客户端
		err       error          //错误接口
	)
	responseBody := ResponseBody{Msg: "success"} //响应成功信息
	defer TimeCost(time.Now(), &responseBody)    //响应耗时计算
	sshInfo := c.PostForm("sshInfo")             //SSH客户端信息
	//取POST提交标识
	id := c.PostForm("id")
	//JSON反序列化sshInfo为SSHClient对象
	if sshClient, err = core.DecodedMsgToSSHClient(sshInfo); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error() //出错，替换错误响应消息
		return &responseBody
	}
	//创建SFTP客户端
	if err := sshClient.CreateSftp(); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error() //出错，替换错误响应消息
		return &responseBody
	}
	defer sshClient.Close() //关闭SFTP客户端
	//HTTP操作文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		responseBody.Msg = err.Error() //出错，替换错误响应消息
		return &responseBody
	}
	defer file.Close() //关闭文件
	//默认路径
	path := strings.TrimSpace(c.DefaultPostForm("path", "/root"))
	pathArr := []string{strings.TrimRight(path, "/")}
	//默认文件夹
	if dir := c.DefaultPostForm("dir", ""); "" != dir {
		pathArr = append(pathArr, dir)
		//创建路径
		if err := sshClient.Mkdirs(strings.Join(pathArr, "/")); err != nil {
			responseBody.Msg = err.Error()
			return &responseBody
		}
	}
	pathArr = append(pathArr, header.Filename) //拼接路径与文件名
	//上传文件到指定路径
	err = sshClient.Upload(file, id, strings.Join(pathArr, "/"))
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
	}
	return &responseBody
}

// DownloadFile 下载文件
func DownloadFile(c *gin.Context) *ResponseBody {
	//局部变量
	var (
		sshClient core.SSHClient //SSH客户端
		err       error          //错误接口
	)
	responseBody := ResponseBody{Msg: "success"} //响应成功消息
	defer TimeCost(time.Now(), &responseBody)    //响应时间计算
	//路径处理
	path := strings.TrimSpace(c.DefaultQuery("path", "/root"))
	//查询SSH信息
	sshInfo := c.DefaultQuery("sshInfo", "")
	//JSON反序列sshInfo为SSHClient对象
	if sshClient, err = core.DecodedMsgToSSHClient(sshInfo); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//创建SFTP客户端
	if err := sshClient.CreateSftp(); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer sshClient.Close() //关闭SSH客户端
	//下载文件
	if sftpFile, err := sshClient.Download(path); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
	} else {
		defer sftpFile.Close()              //关闭SFTP客户端
		c.Writer.WriteHeader(http.StatusOK) //写HTTP状态码
		fileMeta := strings.Split(path, "/")
		//设置HTTP头
		c.Header("Content-Disposition", "attachment; filename="+fileMeta[len(fileMeta)-1])
		_, _ = io.Copy(c.Writer, sftpFile) //复制文件到客户端
	}
	return &responseBody
}

// UploadProgressWs 获取上传进度ws
func UploadProgressWs(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"} //响应成功信息
	defer TimeCost(time.Now(), &responseBody)    //响应耗时计算
	//更新websocket响应
	wsConn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	//更新失败
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//取ID
	id := c.Query("id")
	var ready, find bool
	//遍历
	for {
		//未有任何请求响应操作继续循环
		if !ready && core.WcList == nil {
			continue
		}
		//遍历websocket连接池
		for _, v := range core.WcList {
			//如果标识匹配
			if v.Id == id {
				//向指定websocket连接发送进度消息
				wsConn.WriteMessage(1, []byte(strconv.Itoa(v.Total)))
				find = true //已找到连接
				if !ready {
					ready = true //已就绪
				}
				break //跳出遍历
			}
		}
		//已成功发送进度，跳出循环
		if ready && !find {
			wsConn.Close() //关闭websocket连接
			break
		}

		if ready {
			time.Sleep(300 * time.Millisecond) //休眠300毫秒
			find = false
		}
	}
	return &responseBody
}

// FileList 获取文件列表
func FileList(c *gin.Context) *ResponseBody {
	responseBody := ResponseBody{Msg: "success"}          //响应成功消息，处理Msg字段
	defer TimeCost(time.Now(), &responseBody)             //响应耗时计算，处理Duration字段
	path := c.DefaultQuery("path", "/root")               //路径查找
	sshInfo := c.DefaultQuery("sshInfo", "")              //SSH客户端信息查找
	sshClient, err := core.DecodedMsgToSSHClient(sshInfo) //SSH客户端信息JSON反序列化为SSHClient
	//JSON反序列化失败
	if err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	//创建SFTP客户端
	if err := sshClient.CreateSftp(); err != nil {
		fmt.Println(err)
		responseBody.Msg = err.Error()
		return &responseBody
	}
	defer sshClient.Close()                    //关闭SFTP客户端
	files, err := sshClient.Sftp.ReadDir(path) //读取指定路径文件与文件夹列表
	//读取文件夹失败w
	if err != nil {
		if strings.Contains(err.Error(), "exist") {
			//指定路径下没有文件或文件夹
			responseBody.Msg = fmt.Sprintf("Directory %s: no such file or directory", path)
		} else {
			responseBody.Msg = err.Error()
		}
		return &responseBody
	}
	var (
		fileList fileSplice //文件列表
		fileSize string     //文件大小
	)
	//遍历文件列表
	for _, mFile := range files {
		if mFile.IsDir() { //处理文件夹
			fileSize = strconv.FormatInt(mFile.Size(), 10)
		} else { //处理文件
			fileSize = Bytefmt(uint64(mFile.Size()))
		}
		//构造文件信息
		file := File{Name: mFile.Name(),
			IsDir:      mFile.IsDir(),
			Size:       fileSize,
			ModifyTime: mFile.ModTime().Format("2006-01-02 15:04:05")}
		fileList = append(fileList, file) //添加到文件列表
	}
	sort.Stable(fileList) //排序
	//响应数据构造，处理Data字段
	responseBody.Data = map[string]interface{}{
		"path": path,     //路径
		"list": fileList, //文件列表
	}
	return &responseBody //返回HTTP响应结构体
}
