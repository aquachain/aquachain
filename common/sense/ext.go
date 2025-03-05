package sense

import (
	"fmt"
	"os"
	"strings"
)

var main_argv = os.Args // allow test package to override

// FeatureEnabled returns true if the os env is truthy, or flagname is found in command line
func FeatureEnabled(envname string, flagname string) bool {
	if envname == "" && flagname == "" {
		panic("FeatureEnabled called with no args")
	}
	if envname != "" {
		if EnvBool(envname) {
			return true
		}
	}
	if flagname != "" {
		if FastParseArgsBool(flagname) {
			return true
		}
	}
	return false
}

// FeatureDisabled returns true if the os env is falsy, or flagname is found in command line
func FeatureDisabled(envname string, flagname string) bool {
	if envname == "" && flagname == "" {
		panic("FeatureDisabled called with no args")
	}
	if envname != "" {
		if EnvBoolDisabled(envname) {
			return true
		}
	}
	if flagname != "" {
		if FastParseArgsBool(flagname) {
			return true
		}
	}
	return false
}

// FastParseArgs is a quick way to check if a flag has been FOUND on the actual command line
//
// returns true if flagname is found, and the next argument
// if there is one.
//
// example:
//
//	found, value := FastParseArgs("-flagname")
//
//	if found {
//		// do something with value
//	}
//
// Completely skips first arg for tests
func FastParseArgs(flagname string) (bool, string) {
	if strings.Contains(flagname, "-") {
		panic("here, flagname should not contain -")
	}
	argc := len(main_argv)
	if argc == 0 {
		panic("no args")
	}
	argv := main_argv
	for i := 1; i < argc; i++ {
		if strings.Contains(argv[i], "=") {
			if strings.Split(argv[i], "=")[0] == "-"+flagname {
				return true, strings.Split(argv[i], "=")[1]
			}
		}
		if strings.Replace(argv[i], "-", "", 2) == flagname {
			if i+1 < argc {
				return true, argv[i+1]
			}
			return true, ""
		}
	}
	return false, ""
}

// FastParseArgsBool is a quick way to check if a bool flag has been FOUND on the actual command line
func FastParseArgsBool(flagname string) bool {
	x, next := FastParseArgs(flagname)
	if next == "" {
		return x
	}
	// next is "disabled"
	if isFalsy(next) {
		fmt.Fprintf(os.Stderr, "warn: bool is falsy, right?!: %q\n", next)
		return false
	}
	return x
}

func boolString(s string, unset bool, unparsable bool) bool {
	switch strings.ToLower(s) {
	case "":
		return unset
	case "true", "yes", "1", "on", "enabled", "enable":
		return true
	case "false", "no", "0", "off", "disabled", "disable":
		return false
	default:
		fmt.Fprintf(os.Stderr, "warn: unknown bool string: %q\n", s)
		return unparsable
	}

}

// EnvBool returns false if empty/unset/falsy, true if otherwise non-empty
func EnvBool(name string) bool {
	x, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	return boolString(x, false, true)
}

// EnvBoolDisabled returns true only if nonempty+falsy (such as "0" or "false")
//
// a bit different logic than !EnvBool
func EnvBoolDisabled(name string) bool {
	x, ok := os.LookupEnv(name)
	if !ok {
		return false
	}
	return isFalsy(x)
}

func isFalsy(s string) bool {
	return !boolString(s, true, true)
}

// EnvOr returns the value of the environment variable, or the default if unset
func EnvOr(name, def string) string {
	x, ok := os.LookupEnv(name)
	if !ok {
		return def
	}
	return x
}
