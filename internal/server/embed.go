package server

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:dist
var embeddedFS embed.FS

// FileServerHandler returns an HTTP handler serving embedded static frontend assets.
func FileServerHandler() http.Handler {
	distFS, err := fs.Sub(embeddedFS, "dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(distFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/")

		// If root or empty, serve index.html
		if path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Try opening the file
		f, err := distFS.Open(path)
		if err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// For SPA navigation fallback to index.html
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
