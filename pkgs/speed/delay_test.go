package speed

import (
	"fmt"
	"testing"
)

func TestCFSpeedTest_TestDelayOnce(t *testing.T) {

	st := &CFSpeedTest{
		EnableTLS:        true,
		SpeedTestTimeout: 5,
		MinSpeed:         4,
		DelayTestURL:     "www.visa.com.hk/cdn-cgi/trace",
	}

	locationMap := st.GetLocationMap()
	result, err := st.TestDelayOnce(IpPair{ip: "216.116.134.221", port: 2053}, locationMap)
	fmt.Println(result, err)
}
