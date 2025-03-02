package sense

import "testing"

func TestEnv1(t *testing.T) {
	var nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
	if nokeysmode {
		println("test nokeys mode was found")
	}

}
