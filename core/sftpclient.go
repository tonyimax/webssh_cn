// Package core : 核心包
package core

import (
	"github.com/pkg/sftp" //sftp包
	"io"                  //io操作包
	"mime/multipart"      //MIME多部分解析
	"os"                  //操作系统信息
)

// CreateSftp 创建sftp客户端
func (sclient *SSHClient) CreateSftp() error {
	err := sclient.GenerateClient() //生成SSH客户端
	if err != nil {
		return err
	}
	client, err := sftp.NewClient(sclient.Client) //创建SFTP客户端
	if err != nil {
		return err
	}
	sclient.Sftp = client //保存生成的SFTP客户端
	return nil
}

// Mkdirs 创建文件夹
// path 文件夹路径名
func (sclient *SSHClient) Mkdirs(path string) error {
	//如果路径不存在
	if _, err := sclient.Sftp.Stat(path); os.IsNotExist(err) {
		return sclient.Sftp.MkdirAll(path) //根据路径创建文件夹
	}
	return nil
}

// Download 下载文件
// srcPath 文件路径
func (sclient *SSHClient) Download(srcPath string) (*sftp.File, error) {
	return sclient.Sftp.Open(srcPath) //打开文件，返回sftp.File结构的指针与错误接口
}

// Upload 上传文件
// file : mime文件对象
// id: 上传标识
// dstPath : 目标路径
func (sclient *SSHClient) Upload(file multipart.File, id, dstPath string) error {
	dstFile, err := sclient.Sftp.Create(dstPath) //创建目标路径
	if err != nil {
		return err
	}
	defer dstFile.Close() //上传成功后关闭文件
	defer func() {
		// 上传完后删掉slice里面的数据
		if len(WcList) < 2 {
			WcList = nil
		} else {
			for i := 0; i < len(WcList); i++ {
				if WcList[i].Id == id {
					WcList = append(WcList[:i], WcList[i+1:]...)
					break
				}
			}
		}
	}()
	wc := WriteCounter{Id: id}                         //更新写计数器
	WcList = append(WcList, &wc)                       //更新websocket列表
	_, err = io.Copy(dstFile, io.TeeReader(file, &wc)) //复制文件到服务器
	if err != nil {
		return err
	}
	return nil
}
