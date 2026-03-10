package server

import (
	_ "embed"
	"net/http"
)

//go:embed install.sh
var installScript []byte

// InstallScriptHandler serves the CLI install script at /install.sh.
func InstallScriptHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/x-shellscript; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write(installScript)
	})
}
