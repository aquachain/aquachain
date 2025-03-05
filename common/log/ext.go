package log

import (
	"time"

	"gitlab.com/aquachain/aquachain/common/sense"
)

var NoSync = !sense.EnvBoolDisabled("NO_LOGSYNC")

func init() {
	go func() {
		time.Sleep(time.Second * 2)
		println("log nosync:", NoSync)
	}()
}
