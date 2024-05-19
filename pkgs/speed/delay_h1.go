package speed

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

func (st *CFSpeedTest) TestDelayUseH1(ipPair IpPair) (*DelayResult, error) {
	dialer := &net.Dialer{
		Timeout:   timeout,
		KeepAlive: 0,
	}
	start := time.Now()
	conn, err := dialer.Dial("tcp", net.JoinHostPort(ipPair.ip, strconv.Itoa(ipPair.port)))
	if err != nil {
		if st.VerboseMode {
			fmt.Printf("connect failed, ip: %s err: %s\n", ipPair.String(), err)
		}
		return nil, err
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(maxDuration))

	tcpDuration := time.Since(start)
	start = time.Now()

	client := http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // 跳过证书验证
			Dial: func(network, addr string) (net.Conn, error) {
				return conn, nil
			},
		},
		Timeout: timeout,
	}

	requestURL := st.GetDelayTestURL()
	req, _ := http.NewRequest("GET", requestURL, nil)

	// 添加用户代理
	req.Header.Set("User-Agent", UA)
	req.Close = true
	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()
	resp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		if st.VerboseMode {
			fmt.Printf("http request failed, ip: %s err: %s\n", ipPair.String(), err)
		}
		return nil, err
	}
	defer resp.Body.Close()
	duration := time.Since(start)
	if duration > maxDuration {
		err := fmt.Errorf("timeout")
		return nil, err
	}
	body, err := st.readWithTimeout(resp, maxDuration)
	if err != nil {
		return nil, err
	}
	return &DelayResult{
		duration: tcpDuration,
		body:     string(body),
	}, nil
}

func (st *CFSpeedTest) readWithTimeout(resp *http.Response, tt time.Duration) ([]byte, error) {
	buf := &bytes.Buffer{}
	timeout := time.After(tt)
	// 使用一个 goroutine 来读取响应体，可能会残留gorutine？
	done := make(chan error)
	go func() {
		_, err := io.Copy(buf, resp.Body)
		done <- err
	}()
	// 等待读取操作完成或者超时
	select {
	case err := <-done:
		return buf.Bytes(), err
	case <-timeout:
		// 读取操作超时
		return nil, fmt.Errorf("read timeout")
	}

}
