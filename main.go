package main

import (
	"flag"
	"fmt"
	asn2 "github.com/jackrun123/cfiptest/pkgs/asn"
	"github.com/jackrun123/cfiptest/pkgs/speed"
	"math/rand"
	"os"
	"time"
)

var (
	version      string
	printVersion bool
	isShowHelp   bool
	st           = speed.CFSpeedTest{}
	asn          = asn2.ASN{}
	asnCmd       *flag.FlagSet
)

func init() {
	rand.Seed(time.Now().Unix())
	flag.StringVar(&st.IpFile, "f", "ip.txt", "IP地址文件名称，格式1.0.0.127,443")
	flag.StringVar(&st.OutFile, "o", "ip.csv", "输出文件名称")
	flag.IntVar(&st.DefaultPort, "p", 443, "默认端口")
	flag.IntVar(&st.MaxThread, "dt", 100, "并发请求最大协程数")
	flag.IntVar(&st.SpeedTestTimeout, "sto", 5, "速度测试超时时间")
	flag.IntVar(&st.SpeedTestThread, "st", 1, "下载测速协程数量,设为0禁用测速")
	flag.StringVar(&st.SpeedTestURL, "url", "speed.cloudflare.com/__down?bytes=100000000", "测速文件地址")
	flag.StringVar(&st.DelayTestURL, "delay_url", "www.visa.com.hk", "延迟测试地址，要求是使用cloudflare的地址，只用填域名")
	flag.IntVar(&st.DelayTestType, "dtt", 0, "延迟测试类型, 0: http测试 1：tcp测试")
	flag.IntVar(&st.MaxSpeedTestCount, "maxsc", 10, "速度测试，最多测试多少个IP")
	flag.IntVar(&st.MaxDelayCount, "maxdc", 0, "延迟测试，最多测试多少个IP，如果不限制则设置为0")
	flag.Float64Var(&st.MinSpeed, "mins", 1, "最低速度")
	flag.BoolVar(&st.EnableTLS, "tls", true, "是否启用TLS")
	flag.BoolVar(&st.Shuffle, "s", false, "是否打乱顺序测速")
	flag.StringVar(&st.FilterIATA, "iata", "", "使用IATA过滤，多个用英文逗号分隔，例如：HKG,SIN")
	flag.BoolVar(&st.TestWebSocket, "w", false, "是否验证websocket，如果要验证，delay_url需要支持websocket，客户端会请求xx.com/ws地址")
	flag.BoolVar(&st.VerboseMode, "vv", false, "详细日志模式，打印出错信息")
	flag.BoolVar(&printVersion, "v", false, "打印程序版本")
	flag.BoolVar(&isShowHelp, "h", false, "帮助")

	asnCmd = flag.NewFlagSet("asn", flag.ExitOnError)
	asnCmd.StringVar(&asn.AsCode, "as", "", "ASN号码，例如13335")
}

func main() {
	cmd := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}

	switch cmd {
	case "asn":
		asnCmd.Parse(os.Args[2:])
		asn.Run()
	default:
		flag.Usage = func() {
			fmt.Println("使用方法：")
			fmt.Println("例子：cfiptest -f ./ip.txt -url speed.cloudflare.com/__down?bytes=100000000")
			fmt.Println("参数：")
			flag.PrintDefaults()
			fmt.Println()
			fmt.Println("cfiptest asn 用于根据asn获取ip段")
			fmt.Println("例子：cfiptest asn -as 13335")
			asnCmd.PrintDefaults()

		}
		flag.Parse()
		if printVersion {
			println(version)
			os.Exit(0)
		}

		if isShowHelp {
			flag.Usage()
			os.Exit(0)
		}
		st.Run()
	}
}
