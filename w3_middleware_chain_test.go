package libw3_test

import (
	"testing"

	libw3 "codeberg.org/bbfh/lib-w3"
	"codeberg.org/bbfh/lib-w3/w3ware"
	"gotest.tools/assert"
)

func TestNames(t *testing.T) {
	chain := libw3.MiddlewareChain{
		w3ware.MustLogIn,
		w3ware.Recovery,
		w3ware.RequestLogger,
		&w3ware.CsrfProtection{},
		&w3ware.SessionAuth[string]{},
	}
	assert.DeepEqual(t, chain.Names(), []string{
		"w3ware.MustLogIn",
		"w3ware.Recovery",
		"w3ware.RequestLogger",
		"w3ware.CsrfProtection",
		"w3ware.SessionAuth[string]",
	})
}
