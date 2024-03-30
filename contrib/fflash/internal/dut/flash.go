// Copyright 2022 The ChromiumOS Authors
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package dut

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"syscall"

	"cloud.google.com/go/storage"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/readahead"
	"golang.org/x/oauth2"
	"google.golang.org/api/option"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/misc"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/progress"
)

// copyChunked copies r to w in chunks.
func copyChunked(w io.Writer, r io.Reader, buf []byte) (written int64, err error) {
	for {
		n, err := io.ReadFull(r, buf)
		if err != nil && err != io.ErrUnexpectedEOF {
			if err == io.EOF {
				break
			}
			return written, err
		}
		if m, err := w.Write(buf[:n]); err != nil {
			return written, err
		} else {
			written += int64(m)
		}
	}
	return written, nil
}

type closeFunc func() error

// Client creates a storage.Client from req.
func (req *Request) Client(ctx context.Context) (*storage.Client, error) {
	client, err := storage.NewClient(ctx,
		option.WithTokenSource(oauth2.StaticTokenSource(req.Token)),
	)
	if err != nil {
		return nil, fmt.Errorf("storage.NewClient failed: %w", err)
	}

	return client, nil
}

// objectHandle returns the storage.ObjectHandle for the artifact bucket, dir and name.
func objectHandle(client *storage.Client, bucket, dir, name string) *storage.ObjectHandle {
	return client.Bucket(bucket).Object(path.Join(dir, name))
}

// openObject opens the file in the the directory specified by req.
func (req *Request) openObject(ctx context.Context, client *storage.Client, rw *progress.ReportingWriter, name string, decompress bool) (io.Reader, closeFunc, error) {
	obj := objectHandle(client, req.Artifact.Bucket, req.Artifact.Dir, name)

	rd, err := obj.NewReader(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("obj.NewReader for %s failed: %w", misc.GsURI(obj), err)
	}
	rw.SetTotal(rd.Attrs.Size)

	aheadRd, err := readahead.NewReadCloserSize(rd, 4, 1<<20)
	if err != nil {
		rd.Close()
		return nil, nil, fmt.Errorf(
			"readahead.NewReadCloserSize for %s failed: %w", misc.GsURI(obj), err)
	}

	brd := io.TeeReader(aheadRd, rw)

	var decompressRd io.ReadCloser
	if decompress {
		switch req.Artifact.Compression {
		case artifact.Gzip:
			decompressRd, err = gzip.NewReader(brd)
			if err != nil {
				rd.Close()
				return nil, nil, fmt.Errorf("gzip.NewReader for %s failed: %w", misc.GsURI(obj), err)
			}
		case artifact.Zstd:
			var d *zstd.Decoder
			d, err = zstd.NewReader(brd)
			if err != nil {
				rd.Close()
				return nil, nil, fmt.Errorf("zstd.NewReader for %s failed: %w", misc.GsURI(obj), err)
			}
			decompressRd = d.IOReadCloser()
		default:
			log.Panicf("unknown compression %q", req.Artifact.Compression)
		}
	} else {
		decompressRd = io.NopCloser(brd)
	}

	return decompressRd,
		func() error {
			decompressRd.Close()
			return aheadRd.Close()
		},
		nil
}

// CheckArtifact checks that all image artifacts are accessible.
func CheckArtifact(ctx context.Context, client *storage.Client, artifact *artifact.Artifact) error {
	for _, file := range artifact.Images.Names() {
		obj := objectHandle(client, artifact.Bucket, artifact.Dir, file)

		if _, err := obj.Attrs(ctx); err != nil {
			return fmt.Errorf("%s: %w", misc.GsURI(obj), err)
		}
	}
	return nil
}

// Flash a partition with imageGz to partition.
func (req *Request) Flash(ctx context.Context, client *storage.Client, rw *progress.ReportingWriter, image string, partition string) error {
	r, close, err := req.openObject(ctx, client, rw, image, true)
	if err != nil {
		return err
	}
	defer close()

	w, err := os.OpenFile(partition, os.O_WRONLY|syscall.O_DIRECT, 0660)
	if err != nil {
		return fmt.Errorf("cannot open %s: %w", partition, err)
	}
	defer func() {
		if err := w.Close(); err != nil {
			panic(err)
		}
	}()

	if _, err := copyChunked(w, r, make([]byte, 1<<20)); err != nil {
		return fmt.Errorf("copy to %s failed: %w", partition, err)
	}

	return nil
}

// FlashStateful flashes the stateful partition.
func (req *Request) FlashStateful(ctx context.Context, client *storage.Client, rw *progress.ReportingWriter, clobber bool) error {
	r, close, err := req.openObject(ctx, client, rw, req.Artifact.Stateful, false)
	if err != nil {
		return err
	}
	defer close()

	if err := unpackStateful(ctx, r, req.Artifact.Compression); err != nil {
		return err
	}

	content := ""
	if clobber {
		content = "clobber"
	}

	if err := os.WriteFile(
		filepath.Join(statefulDir, statefulAvailable),
		[]byte(content),
		0644,
	); err != nil {
		return fmt.Errorf("failed to write %s: %w", statefulAvailable, err)
	}

	return nil
}

// RunPostinst runs "postinst" from the partition.
func RunPostinst(ctx context.Context, partition string) error {
	dir, err := os.MkdirTemp("/tmp", "dut-agent-*")
	if err != nil {
		return err
	}

	if _, err := runCommand(ctx, "mount", "-o", "ro", partition, dir); err != nil {
		return err
	}
	defer func() {
		if _, err := runCommand(context.Background(), "umount", partition); err != nil {
			log.Printf("failed to unmount rootfs: %s", err)
		}
	}()

	return runCommandStderr(ctx, filepath.Join(dir, "postinst"), partition)
}

func DisableRootfsVerification(ctx context.Context, kernelNum int) error {
	return runCommandStderr(ctx,
		"/usr/share/vboot/bin/make_dev_ssd.sh",
		"--remove_rootfs_verification",
		"--partitions", strconv.Itoa(kernelNum),
	)
}

func ClearTpmOwner(ctx context.Context) error {
	return runCommandStderr(ctx,
		"crossystem",
		"clear_tpm_owner_request=1",
	)
}
