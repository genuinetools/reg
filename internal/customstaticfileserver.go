package internal

import (
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path"
	"strings"
)

// CustomStaticFileServer serves files and fallback to the given handler if the file does not exists
// https://siongui.github.io/2017/03/19/go-file-server-with-custom-404-not-found/
func CustomStaticFileServer(root http.Dir, NotFoundHandler http.Handler) http.Handler {
	rootPath := string(root)
	logrus.Infof("Looking for static files in %s", rootPath)
	fsh := http.FileServer(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := path.Clean(strings.TrimPrefix(r.URL.Path, "/static/"))
		logrus.Debugf("opening file %s in %s", path, rootPath)
		f, err := root.Open(path)
		if os.IsNotExist(err) {
			NotFoundHandler.ServeHTTP(w, r)
			return
		}
		fsh.ServeHTTP(w, r)
		defer f.Close()
	})
}
