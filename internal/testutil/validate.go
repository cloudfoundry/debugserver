package testutil

import (
	"code.cloudfoundry.org/debugserver"
	"net/http"
)

// Exported only for tests
func ValidateAndNormalize(w http.ResponseWriter, r *http.Request, level []byte) (string, error) {
	return debugserver.ValidateAndNormalize(w, r, level)
}
