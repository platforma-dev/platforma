package httpserver

import (
	"context"
	"io/fs"
	"net/http"
	"time"
)

// FileServer represents an HTTP file server for serving static files.
type FileServer struct {
	server *HTTPServer
}

// NewFileServer creates a new FileServer instance with the given file system, base path, and port.
func NewFileServer(fs fs.FS, basePath, port string) *FileServer {
	server := New(port, 1*time.Second)
	server.Mount(basePath, http.FileServer(http.FS(fs)))

	return &FileServer{server: server}
}

func (s *FileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.server.ServeHTTP(w, r)
}

// Run starts the file server and listens for incoming requests.
func (s *FileServer) Run(ctx context.Context) error {
	return s.server.Run(ctx)
}
