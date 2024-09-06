#!/bin/bash
#github平台上生成的用户令牌
github_token=""
#github项目名
project="Jrohy/webssh"
#获取当前的这个脚本所在绝对路径
shell_path=$(cd `dirname $0`; pwd)
#取发布id号：140461653
release_id=`curl -H 'Cache-Control: no-cache' -s https://api.github.com/repos/$project/releases/latest|grep id|awk 'NR==1{print $2}'|sed 's/,//'`

#上传文件函数
function uploadfile() {
    file=$1 #获取文件路径
    ctype=$(file -b --mime-type $file) #获取文件mime类型
    #上传到github, basename用于取传入路径的文件名
    curl -H "Authorization: token ${github_token}" -H "Content-Type: ${ctype}" --data-binary @$file "https://uploads.github.com/repos/$project/releases/${release_id}/assets?name=$(basename $file)"
    echo ""
}

#上传
function upload() {
    file=$1 #要上传的文件
    #验证文件dgst
    dgst=$1.dgst
    openssl dgst -md5 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha1 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha256 $file | sed 's/([^)]*)//g' >> $dgst
    openssl dgst -sha512 $file | sed 's/([^)]*)//g' >> $dgst
    uploadfile $file #上传文件
    uploadfile $dgst #上传dgst
}
# 取tag : e6f408843ead0ff700442676e3e0e307625454df ---> git rev-list --tags --max-count=1
# 取版本号：v1.27  ---> git describe --tags e6f408843ead0ff700442676e3e0e307625454df
version=`git describe --tags $(git rev-list --tags --max-count=1)`
now=`TZ=Asia/Shanghai date "+%Y%m%d-%H%M"` #取日期
go_version=`go version|awk '{print $3,$4}'` #go版本号
git_version=`git rev-parse HEAD` #取git版本号
#链接标志
ldflags="-w -s -X 'main.version=$version' -X 'main.buildDate=$now' -X 'main.goVersion=$go_version' -X 'main.gitVersion=$git_version'"
#生成指定平台可执行程序
#windows平台
GOOS=windows GOARCH=amd64 go build -ldflags "$ldflags" -o result/webssh_windows_amd64.exe .
GOOS=windows GOARCH=386 go build -ldflags "$ldflags" -o result/webssh_windows_386.exe .
#linux平台
GOOS=linux GOARCH=amd64 go build -ldflags "$ldflags" -o result/webssh_linux_amd64 .
GOOS=linux GOARCH=arm64 go build -ldflags "$ldflags" -o result/webssh_linux_arm64 .
#macOS平台
GOOS=darwin GOARCH=amd64 go build -ldflags "$ldflags" -o result/webssh_darwin_amd64 .
GOOS=darwin GOARCH=arm64 go build -ldflags "$ldflags" -o result/webssh_darwin_arm64 .

#进入生成目录
if [[ $# == 0 ]];then
    cd result #进入生成目录
    upload_item=($(ls -l|awk '{print $9}'|xargs -r))
    #遍历并上传生成的二进制文件到github
    for item in ${upload_item[@]}
    do
        upload $item #上传
    done
    echo "upload completed!" #上传完成
    cd $shell_path #回来根目录
    rm -rf result #删除二进制文件目录
fi
