package sense

import (
	"fmt"
	"os"
	"testing"
)

func TestEnv1(t *testing.T) {
	var nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
	if nokeysmode {
		println("test nokeys mode was found")
	} else {
		println("test nokeys mode was not found")
	}
}

func TestEnv2(t *testing.T) {
	restore, shouldRestore := os.LookupEnv("NO_KEYS")
	if shouldRestore {
		defer os.Setenv("NO_KEYS", restore)
	}

	os.Setenv("NO_KEYS", "true")
	{
		main_argv = []string{"env yes flag yes", "-nokeys"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode := FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatal("test nokeys mode was not found")
		}

		main_argv = []string{"env yes flag=no", "-nokeys=false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("env did not override flag")
		}
		main_argv = []string{"env yes flag no", "-nokeys", "false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("env did not override flag")
		}
		main_argv = []string{"env yes flag=yes", "-nokeys=true"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
	}
	os.Unsetenv("NO_KEYS")
	{
		main_argv = []string{"env empty flag yes", "-nokeys"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode := FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}
		main_argv = []string{"env empty flag=true", "-nokeys=true"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}
		main_argv = []string{"env empty flag true", "-nokeys", "true"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}

		main_argv = []string{"env empty flag=false", "-nokeys=false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if nokeysmode {
			t.Fatalf("spurious no keys mode")
		}
		main_argv = []string{"empty empty flag false", "-nokeys", "false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if nokeysmode {
			t.Fatalf("spurious no keys mode")
		}
	}
	os.Setenv("NO_KEYS", "0")
	{
		main_argv = []string{"env 0 flag yes", "-nokeys"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode := FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}
		main_argv = []string{"env 0 flag=true", "-nokeys=true"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}
		main_argv = []string{"env 0 flag true", "-nokeys", "true"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if !nokeysmode {
			t.Fatalf("empty env overrided flag")
		}

		main_argv = []string{"env 0 flag=false", "-nokeys=false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if nokeysmode {
			t.Fatalf("spurious no keys mode")
		}
		main_argv = []string{"env 0 flag false", "-nokeys", "false"}
		fmt.Fprintf(os.Stderr, "%q\n", main_argv)
		nokeysmode = FeatureEnabled("NO_KEYS", "nokeys")
		if nokeysmode {
			t.Fatalf("spurious no keys mode")
		}
	}

}
