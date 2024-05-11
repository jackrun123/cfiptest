package speed

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/quic-go/qlog"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

func (st *CFSpeedTest) TestDelayUseH3(ipPair IpPair) (*DelayResult, error) {
	start := time.Now()
	tlsConf := &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         st.DelayTestURL,
		NextProtos:         []string{"h3-29", "h3", "hq", "quic"},
	}
	quicConf := &quic.Config{
		Tracer: qlog.DefaultTracer,
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	conn, err := quic.DialAddrEarly(ctx, net.JoinHostPort(ipPair.ip, strconv.Itoa(ipPair.port)), tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("connect err, %s", err)
	}
	defer conn.CloseWithError(0, "")
	tcpDuration := time.Since(start)
	start = time.Now()

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: tlsConf,
		QUICConfig:      quicConf,
		Dial: func(ctx context.Context, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
			return conn, nil
		},
	}
	defer roundTripper.Close()
	client := &http.Client{
		Transport: roundTripper,
	}

	requestURL := st.GetDelayTestURL()
	req, _ := http.NewRequest("GET", requestURL, nil)

	// 添加用户代理
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36")
	req.Close = true
	ctx2, cancel2 := context.WithTimeout(context.Background(), maxDuration)
	defer cancel2()
	resp, err := client.Do(req.WithContext(ctx2))
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
