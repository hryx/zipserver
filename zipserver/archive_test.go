package zipserver

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func testLimits() *ExtractLimits {
	return &ExtractLimits{
		MaxFileSize:       1024 * 1024 * 200,
		MaxTotalSize:      1024 * 1024 * 500,
		MaxNumFiles:       100,
		MaxFileNameLength: 80,
		ExtractionThreads: 4,
	}
}

func emptyConfig() *Config {
	return &Config{
		Bucket:            "testbucket",
		ExtractionThreads: 8,
	}
}

func Test_ExtractOnGCS(t *testing.T) {
	withGoogleCloudStorage(t, func(storage Storage, config *Config) {
		archiver := &Archiver{storage, config}

		r, err := os.Open("/home/leafo/code/go/etlua.zip")
		assert.NoError(t, err)
		defer r.Close()

		err = storage.PutFile(config.Bucket, "zipserver_test/test.zip", r, "application/zip")
		assert.NoError(t, err)

		_, err = archiver.ExtractZip("zipserver_test/test.zip", "zipserver_test/extract", testLimits())
		assert.NoError(t, err)
	})
}

type zipEntry struct {
	name                    string
	outName                 string
	data                    []byte
	expectedMimeType        string
	expectedContentEncoding string
	ignored                 bool
}

type zipLayout struct {
	entries []zipEntry
}

func (zl *zipLayout) Write(t *testing.T, zw *zip.Writer) {
	for _, entry := range zl.entries {
		writer, err := zw.CreateHeader(&zip.FileHeader{
			Name:               entry.name,
			UncompressedSize64: uint64(len(entry.data)),
		})
		assert.NoError(t, err)

		_, err = io.Copy(writer, bytes.NewReader(entry.data))
		assert.NoError(t, err)
	}
}

func (zl *zipLayout) Check(t *testing.T, storage *MemStorage, bucket, prefix string) {
	for _, entry := range zl.entries {
		func() {
			name := entry.name
			if entry.outName != "" {
				name = entry.outName
			}

			path := fmt.Sprintf("%s/%s", prefix, name)
			reader, err := storage.GetFile(bucket, path)
			if entry.ignored {
				assert.Error(t, err)
				assert.True(t, strings.Contains(err.Error(), "object not found"))
				return
			}

			assert.NoError(t, err)

			defer reader.Close()

			data, err := io.ReadAll(reader)
			assert.NoError(t, err)
			assert.EqualValues(t, data, entry.data)

			h, err := storage.getHeaders(bucket, path)
			assert.NoError(t, err)
			assert.EqualValues(t, entry.expectedMimeType, h.Get("content-type"))
			assert.EqualValues(t, "public-read", h.Get("x-goog-acl"))

			if entry.expectedContentEncoding != "" {
				assert.EqualValues(t, entry.expectedContentEncoding, h.Get("content-encoding"))
			}
		}()
	}
}

func Test_ExtractInMemory(t *testing.T) {
	config := emptyConfig()

	storage, err := NewMemStorage()
	assert.NoError(t, err)

	archiver := &Archiver{storage, config}
	prefix := "zipserver_test/mem_test_extracted"
	zipPath := "mem_test.zip"

	_, err = archiver.ExtractZip(zipPath, prefix, testLimits())
	assert.Error(t, err)

	withZip := func(zl *zipLayout, cb func(zl *zipLayout)) {
		var buf bytes.Buffer

		zw := zip.NewWriter(&buf)

		zl.Write(t, zw)

		err = zw.Close()
		assert.NoError(t, err)

		err = storage.PutFile(config.Bucket, zipPath, bytes.NewReader(buf.Bytes()), "application/octet-stream")
		assert.NoError(t, err)

		cb(zl)
	}

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             "file.txt",
				data:             []byte("Hello there"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "garbage.bin",
				data:             bytes.Repeat([]byte{3, 1, 5, 3, 2, 6, 1, 2, 5, 3, 4, 6, 2}, 20),
				expectedMimeType: "application/octet-stream",
			},
			zipEntry{
				name:             "something.gz",
				data:             []byte{0x1F, 0x8B, 0x08, 1, 5, 2, 4, 9, 3, 1, 2, 5},
				expectedMimeType: "application/gzip",
			},
			zipEntry{
				name:                    "something.unityweb",
				data:                    []byte{0x1F, 0x8B, 0x08, 9, 1, 5, 2, 3, 5, 2, 6, 4, 4},
				expectedMimeType:        "application/octet-stream",
				expectedContentEncoding: "gzip",
			},
			zipEntry{
				name:                    "gamedata.memgz",
				outName:                 "gamedata.mem",
				data:                    []byte{0x1F, 0x8B, 0x08, 1, 5, 2, 3, 1, 2, 1, 2},
				expectedMimeType:        "application/octet-stream",
				expectedContentEncoding: "gzip",
			},
			zipEntry{
				name:                    "gamedata.jsgz",
				outName:                 "gamedata.js",
				data:                    []byte{0x1F, 0x8B, 0x08, 3, 7, 3, 4, 12, 53, 26, 34},
				expectedMimeType:        "application/octet-stream",
				expectedContentEncoding: "gzip",
			},
			zipEntry{
				name:                    "gamedata.asm.jsgz",
				outName:                 "gamedata.asm.js",
				data:                    []byte{0x1F, 0x8B, 0x08, 62, 34, 128, 37, 10, 39, 82},
				expectedMimeType:        "application/octet-stream",
				expectedContentEncoding: "gzip",
			},
			zipEntry{
				name:                    "gamedata.datagz",
				outName:                 "gamedata.data",
				data:                    []byte{0x1F, 0x8B, 0x08, 8, 5, 23, 1, 25, 38},
				expectedMimeType:        "application/octet-stream",
				expectedContentEncoding: "gzip",
			},
			zipEntry{
				name:    "__MACOSX/hello",
				data:    []byte{},
				ignored: true,
			},
			zipEntry{
				name:    "/woops/hi/im/absolute",
				data:    []byte{},
				ignored: true,
			},
			zipEntry{
				name:    "oh/hey/im/a/dir/",
				data:    []byte{},
				ignored: true,
			},
			zipEntry{
				name:    "im/trying/to/escape/../../../../../../etc/hosts",
				data:    []byte{},
				ignored: true,
			},
		},
	}, func(zl *zipLayout) {
		_, err := archiver.ExtractZip(zipPath, prefix, testLimits())
		assert.NoError(t, err)

		zl.Check(t, storage, config.Bucket, prefix)
	})

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             strings.Repeat("x", 101),
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
		},
	}, func(zl *zipLayout) {
		limits := testLimits()
		limits.MaxFileNameLength = 100

		_, err := archiver.ExtractZip(zipPath, prefix, limits)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "paths that are too long"))
	})

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             "x",
				data:             bytes.Repeat([]byte("oh no"), 100),
				expectedMimeType: "text/plain; charset=utf-8",
			},
		},
	}, func(zl *zipLayout) {
		limits := testLimits()
		limits.MaxFileSize = 499

		_, err := archiver.ExtractZip(zipPath, prefix, limits)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "file that is too large"))
	})

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             "1",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "2",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "3",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "4",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
		},
	}, func(zl *zipLayout) {
		limits := testLimits()
		limits.MaxNumFiles = 3

		_, err := archiver.ExtractZip(zipPath, prefix, limits)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "Too many files"))
	})

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             "1",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "2",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "3",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "4",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
		},
	}, func(zl *zipLayout) {
		limits := testLimits()
		limits.MaxTotalSize = 6

		_, err := archiver.ExtractZip(zipPath, prefix, limits)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "zip too large"))
	})

	// reset storage for this next test
	storage, err = NewMemStorage()
	assert.NoError(t, err)
	storage.planForFailure(config.Bucket, fmt.Sprintf("%s/%s", prefix, "3"))
	storage.putDelay = 200 * time.Millisecond
	archiver = &Archiver{storage, config}

	withZip(&zipLayout{
		entries: []zipEntry{
			zipEntry{
				name:             "1",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "2",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "3",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
			zipEntry{
				name:             "4",
				data:             []byte("uh oh"),
				expectedMimeType: "text/plain; charset=utf-8",
			},
		},
	}, func(zl *zipLayout) {
		limits := testLimits()

		_, err := archiver.ExtractZip(zipPath, prefix, limits)
		assert.Error(t, err)
		assert.True(t, strings.Contains(err.Error(), "intentional failure"))

		assert.EqualValues(t, 1, len(storage.objects), "make sure all objects have been cleaned up")
		for k := range storage.objects {
			assert.EqualValues(t, k, storage.objectPath(config.Bucket, zipPath), "make sure the only remaining object is the zip")
		}
	})
}
