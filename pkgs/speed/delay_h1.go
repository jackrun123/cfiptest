package speed

import (
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
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &DelayResult{
		duration: tcpDuration,
		body:     string(body),
	}, nil
}
