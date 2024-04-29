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
		DelayTestURL:     "www.visa.com.hk",
		TestWebSocket:    true,
	}

	locationMap := st.GetLocationMap()
	result, err := st.TestDelayOnce(IpPair{ip: "8.212.26.41", port: 443}, locationMap)
	fmt.Println(result, err)
}
