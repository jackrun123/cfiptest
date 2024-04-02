package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	requestURL  = "speed.cloudflare.com/cdn-cgi/trace" // 请求trace URL
	timeout     = 1 * time.Second                      // 超时时间
	maxDuration = 2 * time.Second                      // 最大持续时间
)

var (
	version           string
	printVersion      bool
	st                = CFSpeedTest{}
	speedTestTimeout  int
	ipFile            string
	outFile           string
	defaultPort       int
	maxThread         int
	speedTestThread   int
	speedTestURL      string
	maxSpeedTestCount int
	maxDelayCount     int
	minSpeed          float64
	enableTLS         bool
	shuffle           bool
)

func init() {
	rand.Seed(time.Now().Unix())
	flag.StringVar(&ipFile, "f", "ip.txt", "IP地址文件名称，格式1.0.0.127:443")
	flag.StringVar(&outFile, "o", "ip.csv", "输出文件名称")
	flag.IntVar(&defaultPort, "p", 443, "默认端口")
	flag.IntVar(&maxThread, "dt", 100, "并发请求最大协程数")
	flag.IntVar(&speedTestTimeout, "sto", 5, "速度测试超时时间")
	flag.IntVar(&speedTestThread, "st", 1, "下载测速协程数量,设为0禁用测速")
	flag.StringVar(&speedTestURL, "url", "speed.cloudflare.com/__down?bytes=100000000", "测速文件地址")
	flag.IntVar(&maxSpeedTestCount, "maxsc", 10, "速度测试，最多测试多少个IP")
	flag.IntVar(&maxDelayCount, "maxdc", 1000, "延迟测试，最多测试多少个IP")
	flag.Float64Var(&minSpeed, "mins", 1, "最低速度")
	flag.BoolVar(&enableTLS, "tls", true, "是否启用TLS")
	flag.BoolVar(&shuffle, "s", false, "是否打乱顺序测速")
	flag.BoolVar(&printVersion, "v", false, "打印程序版本")
	flag.Parse()

	if printVersion {
		println(version)
		os.Exit(0)
	}
}

type IpPair struct {
	ip   string
	port int
}

func (ip *IpPair) String() string {
	return fmt.Sprintf("%s:%d", ip.ip, ip.port)
}

type Result struct {
	ip          string        // IP地址
	port        int           // 端口
	dataCenter  string        // 数据中心
	region      string        // 地区
	city        string        // 城市
	latency     string        // 延迟
	tcpDuration time.Duration // TCP请求延迟
}

type SpeedTestResult struct {
	Result
	downloadSpeed float64 // 下载速度
}

type Location struct {
	Iata   string  `json:"iata"`
	Lat    float64 `json:"lat"`
	Lon    float64 `json:"lon"`
	Cca2   string  `json:"cca2"`
	Region string  `json:"region"`
	City   string  `json:"city"`
}

type CFSpeedTest struct {
}

func (st *CFSpeedTest) Run() {
	startTime := time.Now()
	locationMap := st.GetLocationMap()
	if locationMap == nil {
		return
	}

	ips, err := st.readIPs(ipFile)
	if err != nil {
		fmt.Printf("无法从文件中读取 IP: %v\n", err)
		return
	}

	if shuffle {
		// 随机顺序
		rand.Shuffle(len(ips), func(i, j int) { ips[i], ips[j] = ips[j], ips[i] })
	}

	resultChan := st.TestDelay(ips, locationMap)
	if len(resultChan) == 0 {
		// 清除输出内容
		fmt.Print("\033[2J")
		fmt.Println("没有发现有效的IP")
		return
	}
	results := st.TestDownload(resultChan)
	st.Output(results)
	// 清除输出内容
	fmt.Print("\033[2J")
	fmt.Printf("成功将结果写入文件 %s，耗时 %d秒\n", outFile, time.Since(startTime)/time.Second)
}

func (st *CFSpeedTest) GetLocationMap() map[string]Location {
	var locations []Location
	if _, err := os.Stat("locations.json"); os.IsNotExist(err) {
		fmt.Println("本地 locations.json 不存在\n正在从 https://speed.cloudflare.com/locations 下载 locations.json")
		resp, err := http.Get("https://speed.cloudflare.com/locations")
		if err != nil {
			fmt.Printf("无法从URL中获取JSON: %v\n", err)
			return nil
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("无法读取响应体: %v\n", err)
			return nil
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("无法解析JSON: %v\n", err)
			return nil
		}
		file, err := os.Create("locations.json")
		if err != nil {
			fmt.Printf("无法创建文件: %v\n", err)
			return nil
		}
		defer file.Close()

		_, err = file.Write(body)
		if err != nil {
			fmt.Printf("无法写入文件: %v\n", err)
			return nil
		}
	} else {
		fmt.Println("本地 locations.json 已存在,无需重新下载")
		file, err := os.Open("locations.json")
		if err != nil {
			fmt.Printf("无法打开文件: %v\n", err)
			return nil
		}
		defer file.Close()

		body, err := ioutil.ReadAll(file)
		if err != nil {
			fmt.Printf("无法读取文件: %v\n", err)
			return nil
		}

		err = json.Unmarshal(body, &locations)
		if err != nil {
			fmt.Printf("无法解析JSON: %v\n", err)
			return nil
		}
	}

	locationMap := make(map[string]Location)
	for _, loc := range locations {
		locationMap[loc.Iata] = loc
	}
	return locationMap
}

// 从文件中读取IP地址
func (cf *CFSpeedTest) readIPs(File string) ([]IpPair, error) {
	file, err := os.Open(File)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var ips []IpPair
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		ipAddr := scanner.Text()
		ip := ipAddr
		port := defaultPort
		// 指定端口
		if strings.Contains(ipAddr, ":") {
			arr := strings.Split(ipAddr, ":")
			ip = arr[0]
			port, _ = strconv.Atoi(arr[1])
		}
		// 判断是否为 CIDR 格式的 IP 地址
		if strings.Contains(ip, "/") {
			ipr, ipNet, err := net.ParseCIDR(ip)
			if err != nil {
				fmt.Printf("无法解析CIDR格式的IP: %v\n", err)
				continue
			}
			for ipn := ipr.Mask(ipNet.Mask); ipNet.Contains(ipn); inc(ipn) {
				ips = append(ips, IpPair{ip: ipn.String(), port: port})
			}
		} else {
			ips = append(ips, IpPair{ip: ip, port: port})
		}
	}
	return ips, scanner.Err()
}

func (st *CFSpeedTest) TestDelay(ips []IpPair, locationMap map[string]Location) chan Result {
	var wg sync.WaitGroup
	wg.Add(len(ips))

	resultChan := make(chan Result, len(ips))

	thread := make(chan struct{}, maxThread)

	count := atomic.Int64{}
	okCount := atomic.Int64{}
	total := len(ips)

	for _, ip := range ips {
		thread <- struct{}{}
		go func(ipPair IpPair) {
			defer func() {
				<-thread
				count.Add(1)
				percentage := float64(count.Load()) / float64(total) * 100
				if count.Load() == int64(total) {
					fmt.Printf("已完成: %d 总数: %d 已完成: %.2f%%\n", count.Load(), total, percentage)
				} else {
					fmt.Printf("已完成: %d 总数: %d 已完成: %.2f%%\r", count.Load(), total, percentage)
				}
				wg.Done()
			}()

			// 如果满足延迟测试条数，则跳过
			if okCount.Load() >= int64(maxDelayCount) {
				return
			}

			dialer := &net.Dialer{
				Timeout:   timeout,
				KeepAlive: 0,
			}
			start := time.Now()
			conn, err := dialer.Dial("tcp", net.JoinHostPort(ipPair.ip, strconv.Itoa(ipPair.port)))
			if err != nil {
				return
			}
			defer conn.Close()

			tcpDuration := time.Since(start)
			start = time.Now()

			client := http.Client{
				Transport: &http.Transport{
					Dial: func(network, addr string) (net.Conn, error) {
						return conn, nil
					},
				},
				Timeout: timeout,
			}

			var protocol string
			if enableTLS {
				protocol = "https://"
			} else {
				protocol = "http://"
			}
			requestURL := protocol + requestURL

			req, _ := http.NewRequest("GET", requestURL, nil)

			// 添加用户代理
			req.Header.Set("User-Agent", "Mozilla/5.0")
			req.Close = true
			ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
			defer cancel()
			resp, err := client.Do(req.WithContext(ctx))
			if err != nil {
				return
			}
			defer resp.Body.Close()
			duration := time.Since(start)
			if duration > maxDuration {
				return
			}
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return
			}

			if strings.Contains(string(body), "uag=Mozilla/5.0") {
				if matches := regexp.MustCompile(`colo=([A-Z]+)`).FindStringSubmatch(string(body)); len(matches) > 1 {
					dataCenter := matches[1]
					loc, ok := locationMap[dataCenter]
					if ok {
						fmt.Printf("发现有效IP %s 位置信息 %s 延迟 %d 毫秒\n", ipPair.String(), loc.City, tcpDuration.Milliseconds())
						resultChan <- Result{ipPair.ip, ipPair.port, dataCenter, loc.Region, loc.City, fmt.Sprintf("%d", tcpDuration.Milliseconds()), tcpDuration}
					} else {
						fmt.Printf("发现有效IP %s 位置信息未知 延迟 %d 毫秒\n", ipPair.String(), tcpDuration.Milliseconds())
						resultChan <- Result{ipPair.ip, ipPair.port, dataCenter, "", "", fmt.Sprintf("%d", tcpDuration.Milliseconds()), tcpDuration}
					}
				}
			}

			okCount.Add(1)
		}(ip)
	}

	wg.Wait()
	close(resultChan)
	return resultChan
}

func (st *CFSpeedTest) TestDownload(resultChan chan Result) []SpeedTestResult {
	var results []SpeedTestResult
	if speedTestThread > 0 {
		fmt.Printf("开始测速，待测速：%d\n", len(resultChan))
		var wg2 sync.WaitGroup
		wg2.Add(speedTestThread)
		count := atomic.Int64{}
		okCount := atomic.Int64{}
		mu := sync.Mutex{}
		total := len(resultChan)
		results = []SpeedTestResult{}
		thread := make(chan struct{}, maxThread)
		for i := 0; i < speedTestThread; i++ {
			thread <- struct{}{}
			go func() {
				defer func() {
					<-thread
					wg2.Done()
				}()
				for res := range resultChan {
					count.Add(1)
					downloadSpeed := getDownloadSpeed(res.ip, res.port)
					mu.Lock()
					if minSpeed <= 0 || downloadSpeed > minSpeed {
						okCount.Add(1)
						results = append(results, SpeedTestResult{Result: res, downloadSpeed: downloadSpeed})
					}
					prefix := fmt.Sprintf("[%d/%d] IP %s:%s ", count.Load(), total, res.ip, strconv.Itoa(res.port))
					if downloadSpeed == -1 {
						fmt.Printf("%s测速无效\n", prefix)
					} else {
						fmt.Printf("%s下载速度 %.2f MB/s, 延迟 %s ms\n", prefix, downloadSpeed, res.latency)
					}

					currentOKCount := okCount.Load()
					percentage := float64(currentOKCount) / float64(total) * 100
					if currentOKCount >= int64(total) || currentOKCount >= int64(maxSpeedTestCount) {
						fmt.Printf("已完成: %d/%d(%.2f%%)，符合条件：%d \u001B[0\n", count.Load(), total, percentage, okCount.Load())
						break
					} else {
						fmt.Printf("已完成: %d/%d(%.2f%%)，符合条件：%d\r", count.Load(), total, percentage, okCount.Load())
					}
					mu.Unlock()
				}
			}()
		}
		wg2.Wait()
	} else {
		for res := range resultChan {
			results = append(results, SpeedTestResult{Result: res})
		}
	}

	if speedTestThread > 0 {
		sort.Slice(results, func(i, j int) bool {
			return results[i].downloadSpeed > results[j].downloadSpeed
		})
	} else {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Result.tcpDuration < results[j].Result.tcpDuration
		})
	}
	return results
}

func (cf *CFSpeedTest) Output(results []SpeedTestResult) {
	file, err := os.Create(outFile)
	if err != nil {
		fmt.Printf("无法创建文件: %v\n", err)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	if speedTestThread > 0 {
		writer.Write([]string{"IP地址", "端口", "TLS", "数据中心", "地区", "城市", "网络延迟(毫秒)", "下载速度(MB/s)"})
	} else {
		writer.Write([]string{"IP地址", "端口", "TLS", "数据中心", "地区", "城市", "网络延迟(毫秒)"})
	}
	for _, res := range results {
		if speedTestThread > 0 {
			writer.Write([]string{res.Result.ip, strconv.Itoa(res.Result.port), strconv.FormatBool(enableTLS), res.Result.dataCenter, res.Result.region, res.Result.city, res.Result.latency, fmt.Sprintf("%.2f", res.downloadSpeed)})
		} else {
			writer.Write([]string{res.Result.ip, strconv.Itoa(res.Result.port), strconv.FormatBool(enableTLS), res.Result.dataCenter, res.Result.region, res.Result.city, res.Result.latency})
		}
	}

	writer.Flush()
}

func main() {
	st.Run()
}

// inc函数实现ip地址自增
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// 测速函数
func getDownloadSpeed(ip string, port int) float64 {
	var protocol string
	if enableTLS {
		protocol = "https://"
	} else {
		protocol = "http://"
	}
	speedTestURL := protocol + speedTestURL
	// 创建请求
	req, _ := http.NewRequest("GET", speedTestURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	// 创建TCP连接
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 0,
	}
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ip, strconv.Itoa(port)))
	if err != nil {
		return -1
	}
	defer conn.Close()

	startTime := time.Now()
	// 创建HTTP客户端
	client := http.Client{
		Transport: &http.Transport{
			Dial: func(network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
		//设置单个IP测速最长时间为5秒
		Timeout: time.Duration(speedTestTimeout) * time.Second,
	}
	// 发送请求
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		return -1
	}
	defer resp.Body.Close()

	// 复制响应体到/dev/null，并计算下载速度
	written, _ := io.Copy(io.Discard, resp.Body)
	duration := time.Since(startTime)
	speed := float64(written) / duration.Seconds() / 1024 / 1024

	return speed
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
