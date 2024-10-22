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
	"path/filepath"
	"strconv"
	"syscall"

	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/artifact"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/payload"
	"chromium.googlesource.com/chromiumos/platform/dev-util.git/contrib/fflash/internal/progress"
	"github.com/klauspost/compress/zstd"
	"github.com/klauspost/readahead"
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

// openObject is a wrapper to open the payload with progress reporting and decompression.
func (req *Request) openObject(ctx context.Context, p payload.Payload, rw *progress.ReportingWriter, imageID artifact.ImageID, decompress bool) (io.Reader, payload.CloseFunc, error) {
	rd, compSize, err := p.OpenImage(ctx, imageID)
	if err != nil {
		return nil, nil, fmt.Errorf("p.OpenObject(): %w", err)
	}
	rw.SetTotal(compSize)

	aheadRd, err := readahead.NewReadCloserSize(rd, 4, 1<<20)
	if err != nil {
		rd.Close()
		return nil, nil, fmt.Errorf(
			"readahead.NewReadCloserSize for %s failed: %w", imageID, err)
	}

	brd := io.TeeReader(aheadRd, rw)

	var decompressRd io.ReadCloser
	if decompress {
		switch p.Compression() {
		case artifact.Gzip:
			decompressRd, err = gzip.NewReader(brd)
			if err != nil {
				rd.Close()
				return nil, nil, fmt.Errorf("gzip.NewReader for %s failed: %w", imageID, err)
			}
		case artifact.Zstd:
			var d *zstd.Decoder
			d, err = zstd.NewReader(brd)
			if err != nil {
				rd.Close()
				return nil, nil, fmt.Errorf("zstd.NewReader for %s failed: %w", imageID, err)
			}
			decompressRd = d.IOReadCloser()
		default:
			log.Panicf("unknown compression %q", p.Compression())
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

// Flash a partition with imageID to partition.
func (req *Request) Flash(ctx context.Context, p payload.Payload, rw *progress.ReportingWriter, imageID artifact.ImageID, partition string, sync bool) error {
	r, close, err := req.openObject(ctx, p, rw, imageID, true)
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

	if sync {
		if err := w.Sync(); err != nil {
			return fmt.Errorf("cannot sync partition: %w", err)
		}

		dropCaches, err := os.OpenFile("/proc/sys/vm/drop_caches", os.O_WRONLY|os.O_TRUNC, 0666)
		if err != nil {
			return fmt.Errorf("failed to open /proc/sys/vm/drop_caches: %w", err)
		}
		defer dropCaches.Close()
		if _, err := dropCaches.WriteString("1\n"); err != nil {
			return fmt.Errorf("failed to write /proc/sys/vm/drop_caches: %w", err)
		}
		if err := dropCaches.Close(); err != nil {
			return fmt.Errorf("failed to close /proc/sys/vm/drop_caches: %w", err)
		}
	}

	return nil
}

// FlashStateful flashes the stateful partition.
func (req *Request) FlashStateful(ctx context.Context, p payload.Payload, rw *progress.ReportingWriter, clobber bool) error {
	r, close, err := req.openObject(ctx, p, rw, artifact.Stateful, false)
	if err != nil {
		return err
	}
	defer close()

	if err := unpackStateful(ctx, r, p.Compression()); err != nil {
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
