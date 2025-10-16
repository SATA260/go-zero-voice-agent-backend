package uniqueid

import "testing"

func TestGenSn(t *testing.T) {
	sn := GenSn(SN_PREFIX_THIRD_PAYMENT)
	t.Log(sn)
}
