package server

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/inspr-at/hausv-org/internal/version"
	"github.com/inspr-at/hausv-org/internal/web"
)

// Embedded files have no modification time. Content ETags give unversioned
// URLs a validator; only the current build's URLs promise immutable content.
func staticAssetHandler() http.Handler {
	assetVersion, _ := url.QueryUnescape(version.AssetVersion())
	var etags sync.Map
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/")
		file, err := web.Assets.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		info, err := file.Stat()
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		content, ok := file.(io.ReadSeeker)
		if !ok {
			http.Error(w, "Asset unavailable", http.StatusInternalServerError)
			return
		}
		etag, ok := etags.Load(name)
		if !ok {
			hash := sha256.New()
			if _, err := io.Copy(hash, content); err != nil {
				http.Error(w, "Asset unavailable", http.StatusInternalServerError)
				return
			}
			etag, _ = etags.LoadOrStore(name, fmt.Sprintf(`"%x"`, hash.Sum(nil)))
		}
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			http.Error(w, "Asset unavailable", http.StatusInternalServerError)
			return
		}
		w.Header().Set("ETag", etag.(string))
		cacheControl := "public, max-age=86400"
		if r.URL.Query().Get("v") == assetVersion {
			cacheControl = "public, max-age=31536000, immutable"
		}
		w.Header().Set("Cache-Control", cacheControl)
		http.ServeContent(w, r, info.Name(), info.ModTime(), content)
	})
}
