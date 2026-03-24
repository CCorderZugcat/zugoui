package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/CCorderZugcat/zugoui/gzasm"
)

func Serve(ctx context.Context, l net.Listener, fsys fs.FS, mux *http.ServeMux) error {
	fshandler := gzasm.New(http.FileServerFS(fsys), fsys)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(fsys, path.Join(".", r.URL.Path)); err == nil {
			fshandler.ServeHTTP(w, r)
			return
		}

		fd, err := fsys.Open("index.html")
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "%v\r\n", err)
			return
		}
		defer fd.Close()

		st, err := fd.Stat()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%v\r\n", err)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Header().Set("Content-Length", strconv.Itoa(int(st.Size())))
		io.Copy(w, fd)
	})

	s := &http.Server{Handler: mux}
	errC := make(chan error)
	defer close(errC)

	go func() {
		err := s.Serve(l)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		errC <- err
	}()

	var err error

	select {
	case <-ctx.Done():
		ctx, stop := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer stop()
		s.Shutdown(ctx)
		err = <-errC

	case err = <-errC:
	}

	return err
}
