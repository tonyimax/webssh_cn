package main //主包名
//导入依赖包
import (
	"embed"                       //可执行文件资源嵌入
	"flag"                        //标志变量
	"fmt"                         //格式化
	"github.com/gin-contrib/gzip" //压缩库
	"github.com/gin-gonic/gin"    //网页框架
	"io/fs"                       //文件系统
	"net/http"                    //http通信
	"os"                          //系统信息
	"strconv"                     //字符串转换
	"strings"                     //字符串
	"time"                        //时间
	"webssh/controller"           //websocket通信
)

// 在可执行文件中嵌入文件夹dist
//
//go:embed web/dist/*
var f embed.FS //集成文件集合，取go:embed中文件夹中文件与文件夹列表
// 变量声明
var (
	//整形标志声明，用于储存整形指针
	port = flag.Int("p", //标志名
		5032,     //标志值
		"服务运行端口") //标志描述
	v        = flag.Bool("v", false, "显示版本号")
	authInfo = flag.String("a", "", "开启账号密码登录验证, '-a user:pass'的格式传参")
	//普通变量声明
	timeout    int    //连接超时
	savePass   bool   //保存密码
	version    string //版本号
	buildDate  string //编译时间
	goVersion  string //go版本号
	gitVersion string //git版本号
	username   string //用户名
	password   string //密码
)

// 初始化
func init() {
	//初始化timeout变量为标志
	flag.IntVar(&timeout, //标志指针
		"t",              //标志名
		120,              //标志值
		"ssh连接超时时间(min)") //标志描述
	//初始化savePass变量为标志
	flag.BoolVar(&savePass, //标志指针
		"s",       //标志名
		true,      //标志值
		"保存ssh密码") //标志描述
	flag.StringVar(&version,
		"ver",
		"v1.0.0",
		"程序版本号")
	flag.StringVar(&goVersion,
		"gover",
		"v1.22",
		"go版本号")
	flag.StringVar(&gitVersion,
		"gitver",
		"2.45.2",
		"git版本号")
	flag.StringVar(&buildDate,
		"d",
		time.Now().String(),
		"编译日期")
	//查找环境变量savePass
	if envVal, ok := os.LookupEnv("savePass"); ok {
		//转换环境变量值为Bool值
		if b, err := strconv.ParseBool(envVal); err == nil {
			savePass = b //如果环境变量有值保存到savePass
		}
	}
	//读取环境变量用户验证信息
	if envVal, ok := os.LookupEnv("authInfo"); ok {
		*authInfo = envVal
	}
	//读取环境变量通信端口信息
	if envVal, ok := os.LookupEnv("port"); ok {
		//转换为整数
		if b, err := strconv.Atoi(envVal); err == nil {
			*port = b
		}
	}
	//必须在标志定义之后及程序访问之前调用
	flag.Parse()
	//如果有-v参数，显示版本号信息
	if *v {
		fmt.Printf("Version: %s\n\n", version)
		fmt.Printf("BuildDate: %s\n\n", buildDate)
		fmt.Printf("GoVersion: %s\n\n", goVersion)
		fmt.Printf("GitVersion: %s\n\n", gitVersion)
		os.Exit(0)
	}
	if *authInfo != "" {
		//分割用户名与密码
		accountInfo := strings.Split(*authInfo, ":")
		//非空判断
		if len(accountInfo) != 2 ||
			accountInfo[0] == "" ||
			accountInfo[1] == "" {
			fmt.Println("请按'user:pass'的格式来传参或设置环境变量, 且账号密码都不能为空!")
			os.Exit(0)
		}
		//保存用户名与密码
		username, password = accountInfo[0], accountInfo[1]
	}
}

// 启动静态路由
func staticRouter(router *gin.Engine) {
	//如果密码不为空
	if password != "" {
		//创建账户列表
		accountList := map[string]string{
			username: password,
		}
		//授权路由
		//传入用户列表{用户:密码}
		authorized := router.Group("/", gin.BasicAuth(accountList))
		authorized.GET("", func(c *gin.Context) {
			//读取主页面
			indexHTML, _ := f.ReadFile("web/dist/" + "index.html")
			//向上下文写入主页面
			c.Writer.Write(indexHTML)
		})
	} else {
		router.GET("/", func(c *gin.Context) {
			indexHTML, _ := f.ReadFile("web/dist/" + "index.html")
			c.Writer.Write(indexHTML)
		})
	}
	staticFs, _ := fs.Sub(f, "web/dist/static")
	router.StaticFS("/static", http.FS(staticFs))
}

func main() {
	server := gin.Default()
	server.SetTrustedProxies(nil)
	server.Use(gzip.Gzip(gzip.DefaultCompression))
	staticRouter(server)
	server.GET("/term", func(c *gin.Context) {
		controller.TermWs(c, time.Duration(timeout)*time.Minute)
	})
	server.GET("/check", func(c *gin.Context) {
		responseBody := controller.CheckSSH(c)
		responseBody.Data = map[string]interface{}{
			"savePass": savePass,
		}
		c.JSON(200, responseBody)
	})
	file := server.Group("/file")
	{
		file.GET("/list", func(c *gin.Context) {
			c.JSON(200, controller.FileList(c))
		})
		file.GET("/download", func(c *gin.Context) {
			controller.DownloadFile(c)
		})
		file.POST("/upload", func(c *gin.Context) {
			c.JSON(200, controller.UploadFile(c))
		})
		file.GET("/progress", func(c *gin.Context) {
			controller.UploadProgressWs(c)
		})
	}
	server.Run(fmt.Sprintf(":%d", *port))
}
