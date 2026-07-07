package sftpd

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"

	"github.com/yourorg/panel-daemon/internal/files"
)

type fsHandler struct {
	baseDir  string
	readOnly bool
}

func (h *fsHandler) resolve(p string) (string, error) {
	return files.SafeJoin(h.baseDir, p)
}

func (h *fsHandler) Fileread(r *sftp.Request) (io.ReaderAt, error) {
	full, err := h.resolve(r.Filepath)
	if err != nil {
		return nil, err
	}
	return os.Open(full)
}

func (h *fsHandler) Filewrite(r *sftp.Request) (io.WriterAt, error) {
	if h.readOnly {
		return nil, sftp.ErrSSHFxPermissionDenied
	}
	full, err := h.resolve(r.Filepath)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(full, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
}

func (h *fsHandler) Filecmd(r *sftp.Request) error {
	if h.readOnly {
		return sftp.ErrSSHFxPermissionDenied
	}
	full, err := h.resolve(r.Filepath)
	if err != nil {
		return err
	}
	switch r.Method {
	case "Setstat":
		return nil
	case "Rename":
		target, err := h.resolve(r.Target)
		if err != nil {
			return err
		}
		return os.Rename(full, target)
	case "Rmdir", "Remove":
		return os.RemoveAll(full)
	case "Mkdir":
		return os.MkdirAll(full, 0755)
	case "Symlink":
		return errors.New("symlinks are not supported")
	}
	return sftp.ErrSSHFxOpUnsupported
}

func (h *fsHandler) Filelist(r *sftp.Request) (sftp.ListerAt, error) {
	full, err := h.resolve(r.Filepath)
	if err != nil {
		return nil, err
	}
	switch r.Method {
	case "List":
		entries, err := os.ReadDir(full)
		if err != nil {
			return nil, err
		}
		infos := make([]os.FileInfo, 0, len(entries))
		for _, e := range entries {
			if info, err := e.Info(); err == nil {
				infos = append(infos, info)
			}
		}
		return listerAt(infos), nil
	case "Stat":
		info, err := os.Stat(full)
		if err != nil {
			return nil, err
		}
		return listerAt([]os.FileInfo{info}), nil
	}
	return nil, sftp.ErrSSHFxOpUnsupported
}

type listerAt []os.FileInfo

func (l listerAt) ListAt(dst []os.FileInfo, offset int64) (int, error) {
	if offset >= int64(len(l)) {
		return 0, io.EOF
	}
	n := copy(dst, l[offset:])
	if n < len(dst) {
		return n, io.EOF
	}
	return n, nil
}
