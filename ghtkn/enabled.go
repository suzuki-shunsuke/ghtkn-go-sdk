package ghtkn

import (
	"errors"
	"os"
	"runtime"

	"github.com/suzuki-shunsuke/ghtkn-go-sdk/ghtkn/internal/config"
)

// InputEnabled configures Enabled.
type InputEnabled struct {
	// Envs is an ordered list of environment variable names checked before
	// GHTKN_ENABLE. The first variable that is set determines the result, which
	// lets a tool embedding ghtkn define its own enable/disable switch (e.g. a
	// tool-specific environment variable). The value must be one of true, 1,
	// false, 0.
	Envs []string
}

// Enabled reports whether ghtkn integration should be enabled.
// The result is resolved in the following order:
//
//  1. the first environment variable named in input.Envs that is set,
//  2. the GHTKN_ENABLE environment variable,
//  3. whether the ghtkn configuration file exists.
//
// For 1 and 2 the value must be one of true, 1, false, 0; any other value
// returns an error. For 3 it returns true when the configuration file exists
// and false when it does not.
func Enabled(input *InputEnabled) (bool, error) {
	return enabled(os.Getenv, input)
}

// enabled is the implementation of Enabled with getEnv injected so the
// environment-variable branches can be tested without mutating the real
// environment.
func enabled(getEnv func(string) string, input *InputEnabled) (bool, error) {
	if input != nil {
		for _, env := range input.Envs {
			a := getEnv(env)
			if a == "" {
				continue
			}
			return checkBoolEnv(a)
		}
	}
	if a := getEnv("GHTKN_ENABLE"); a != "" {
		return checkBoolEnv(a)
	}
	p, err := config.GetPath(getEnv, runtime.GOOS)
	if err != nil {
		return false, err
	}
	if _, err := os.Stat(p); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// checkBoolEnv parses a boolean-ish environment variable value:
// true and 1 are true, false and 0 are false, and anything else is an error.
func checkBoolEnv(env string) (bool, error) {
	if env == "true" || env == "1" {
		return true, nil
	}
	if env == "false" || env == "0" {
		return false, nil
	}
	return false, errors.New("the value must be one of true, 1, false, 0")
}
