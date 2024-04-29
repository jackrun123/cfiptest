# 简介
本项目基于 [badafans/Cloudflare-IP-SpeedTest](https://github.com/badafans/Cloudflare-IP-SpeedTest) 修改，感谢原作者。

Cloudflare IP 测速器是一个使用 Golang 编写的小工具，用于测试一些 Cloudflare 的 IP 地址的延迟和下载速度，并将结果输出到 CSV 文件中。

# 运行
默认测速地址不能正常访问，请使用仓库中的_worker.js在cloudflare的worker或者page上部署，支持websocket和下载测速，可以参考[这个视频](https://www.youtube.com/watch?v=S4AZkvgnmmA)自己搭建一个

准备一个ip.txt文件，内容格式为IP[,端口]，其中端口可以省略，如果省略则使用命令行的默认端口

ip.txt例子
```
127.0.0.1,2053
127.0.0.1/24,2053
2400:cb00:2049:0:33f9:7045:cf64:7d93/120,2053
127.0.0.2
```

在终端中运行以下命令来启动程序：
```
# example.com 是指你自己部署在cf的域名，这里只是例子
./cfiptest -f=ip.txt -mins 5 -url example.com/50m -delay_url example.com
```
请替换参数值以符合您的实际需求。

# 环境变量

可选配置，如果配置了，优先以环境变量为准

| 环境变量  | 备注       |
|-------|----------|
| CFIPTEST_DELAY_TEST_URL | 指定延迟测试地址 |
| CFIPTEST_SPEED_TEST_URL | 指定速度测试地址 |

# 参数说明
可以使用 cfiptest -h 获取使用说明
```
cfiptest -h
使用方法：
例子：cfiptest -f ./ip.txt -url speed.cloudflare.com/__down?bytes=100000000
参数：
  -delay_url string
        延迟测试地址，要求是使用cloudflare的地址，只用填域名 (default "www.visa.com.hk")
  -dt int
        并发请求最大协程数 (default 100)
  -f string
        IP地址文件名称，格式1.0.0.127,443 (default "ip.txt")
  -h    帮助
  -maxdc int
        延迟测试，最多测试多少个IP，如果不限制则设置为0
  -maxsc int
        速度测试，最多测试多少个IP (default 10)
  -mins float
        最低速度 (default 1)
  -o string
        输出文件名称 (default "ip.csv")
  -p int
        默认端口 (default 443)
  -s    是否打乱顺序测速
  -st int
        下载测速协程数量,设为0禁用测速 (default 1)
  -sto int
        速度测试超时时间 (default 5)
  -tls
        是否启用TLS (default true)
  -url string
        测速文件地址 (default "speed.cloudflare.com/__down?bytes=100000000")
  -v    打印程序版本
  -vv
        详细日志模式，打印出错信息
  -w    是否验证websocket，如果要验证，delay_url需要支持websocket，客户端会请求xx.com/ws地址

cfiptest asn 用于根据asn获取ip段
例子：cfiptest asn -as 13335
  -as string
        ASN号码，例如13335
```

# 使用建议
可以先使用`masscan`扫描开放的端口，再使用这个工具二次扫描
```shell
sudo masscan -iL ./ipv6.txt -p 2083 --rate 10000 -oL o.txt
cat o.txt|grep open|awk '{print $4","$3}' > ip.txt
```

# 输出说明
程序将输出每个成功测试的 IP 地址的信息，包括 IP 地址、端口、数据中心、地区、城市、网络延迟和下载速度（如果选择测速）。

程序还会将所有结果写入一个 CSV 文件中。

# 如何选择文件
| 系统      | 架构        | 32/64位 | 文件选择                  | 备注      |
|---------|-----------|--------|-----------------------|---------|
| MacOS   | Intel     | 64     | darwin-amd64.tar.gz   | 苹果      |
| MacOS   | ARM       | 64     | darwin-arm64.tar.gz   | 苹果      |
| Windows | Intel/AMD | 32     | windows-386.zip       | 微软      |
| Windows | Intel/AMD | 64     | windows-amd64.zip     | 微软      |
| Linux   | ARM       | 32     | linux-arm.tar.gz      | 安卓32位机器 |
| Linux   | ARM       | 64     | linux-arm64.tar.gz    | 安卓64位机器 |
| Linux   | Intel/AMD | 32     | linux-386.tar.gz      |         |
| Linux   | Intel/AMD | 64     | linux-amd64.tar.gz    |         |
| Linux   | Mips      | 32     | linux-mips.tar.gz     | 路由器     |
| Linux   | Mips      | 64     | linux-mips64.tar.gz   | 路由器     |
| Linux   | Mipsle    | 32     | linux-mipsle.tar.gz   | 路由器     |
| Linux   | Mipsle    | 64     | linux-mips64le.tar.gz | 路由器     |

# 许可证
The MIT License (MIT)

此处，"软件" 指 Cloudflare IP 测速器。

特此授予非限制性许可证，允许任何人获得本软件副本并自由使用、复制、修改、合并、出版发行、散布、再许可和/或销售本软件的副本，以及将本软件与其它软件捆绑在一起使用。

上述版权声明和本许可声明应包含在本软件的所有副本或主要部分中。

本软件按 "原样" 提供，没有任何形式的明示或暗示保证，包括但不限于适销性保证、特定用途适用性保证和非侵权保证。在任何情况下，作者或版权所有者均不对任何索赔、损害或其他责任负责，无论是在合同、侵权或其他方面，由于或与软件或使用或其他交易中的软件产生或与之相关的操作。
