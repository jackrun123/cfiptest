package speed

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func (st *CFSpeedTest) TestDownload(resultChan chan Result) []SpeedTestResult {
	var results []SpeedTestResult
	if st.SpeedTestThread > 0 {
		fmt.Printf("开始测速，待测速：%d\n", len(resultChan))
		var wg2 sync.WaitGroup
		wg2.Add(st.SpeedTestThread)
		count := atomic.Int64{}
		okCount := atomic.Int64{}
		mu := sync.Mutex{}
		total := len(resultChan)
		results = []SpeedTestResult{}
		thread := make(chan struct{}, st.MaxThread)
		for i := 0; i < st.SpeedTestThread; i++ {
			thread <- struct{}{}
			go func() {
				defer func() {
					<-thread
					wg2.Done()
				}()
				for res := range resultChan {
					count.Add(1)
					downloadSpeed, err := st.getDownloadSpeed(res.ip, res.port)
					mu.Lock()
					if st.MinSpeed <= 0 || downloadSpeed > st.MinSpeed {
						okCount.Add(1)
						results = append(results, SpeedTestResult{Result: res, downloadSpeed: downloadSpeed})
					}
					prefix := fmt.Sprintf("[%d/%d] IP %s ", count.Load(), total, net.JoinHostPort(res.ip, strconv.Itoa(res.port)))
					if err != nil {
						fmt.Printf("%s测速无效, err: %s\n", prefix, err)
					} else {
						fmt.Printf("%s下载速度 %.2f MB/s，延迟 %s ms，地区 %s\n", prefix, downloadSpeed, res.latency, res.city)
					}

					currentOKCount := okCount.Load()
					percentage := float64(count.Load()) / float64(total) * 100
					if currentOKCount >= int64(total) || currentOKCount >= int64(st.MaxSpeedTestCount) {
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

	if st.SpeedTestThread > 0 {
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

// 测速函数
func (st *CFSpeedTest) getDownloadSpeed(ip string, port int) (float64, error) {
	var protocol string
	if st.EnableTLS {
		protocol = "https://"
	} else {
		protocol = "http://"
	}

	speedTestURL := st.SpeedTestURL
	if !strings.HasPrefix(st.SpeedTestURL, "http://") && !strings.HasPrefix(st.SpeedTestURL, "https://") {
		speedTestURL = protocol + st.SpeedTestURL
	}

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
		return -1, err
	}
	defer conn.Close()

	startTime := time.Now()
	// 创建HTTP客户端
	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
			Dial: func(network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
		//设置单个IP测速最长时间为5秒
		Timeout: time.Duration(st.SpeedTestTimeout) * time.Second,
	}
	// 发送请求
	req.Close = true
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()

	stop := make(chan bool)
	var written int64 = 0

	if st.SpeedTestTimeout > 2 {
		// 中途检测下载速度
		go func() {
			ticker := time.NewTicker(1500 * time.Millisecond) // 1.5秒后快速检测一次
			defer ticker.Stop()
			select {
			case <-ticker.C:
				elapsed := time.Since(startTime).Seconds()
				speed := float64(written) / elapsed / 1024 / 1024
				if speed < 0.7*float64(st.MinSpeed) {
					stop <- true
				}
			}
		}()
	}

	// 复制响应体到/dev/null，并计算下载速度
	buf := make([]byte, 8024)

outerLoop:
	for {
		select {
		case <-stop:
			break outerLoop
		default:
			n, err := resp.Body.Read(buf)
			if n > 0 {
				written += int64(n)
			}
			if err != nil {
				break outerLoop
			}
		}
	}

	//written, _ := io.Copy(io.Discard, resp.Body)
	duration := time.Since(startTime)
	speed := float64(written) / duration.Seconds() / 1024 / 1024

	return speed, nil
}
