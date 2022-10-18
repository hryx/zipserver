package zipserver

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dhowden/tag"
	"github.com/go-errors/errors"
)

// MusicAnalyzer uses rules according to music albums.
// FLAC, Ogg, and MP3 files are supported, and all other files are ignored.
// Song metadata is collected and returned.
type MusicAnalyzer struct{}

func (m MusicAnalyzer) Analyze(r io.Reader, key string) (AnalyzeResult, error) {
	res := AnalyzeResult{Key: key}

	// TODO: The music tag library requires a ReadSeeker (understandably),
	// which a Reader does not satisfy. Here is a naive implementation that
	// reads the entire file into memory. To lower memory usage, this should
	// be replaced with a custom ReadSeeker that uses a limited buffer for
	// the unfortunately necessary seeks.

	b, err := io.ReadAll(r)
	if err != nil {
		return res, errors.Wrap(err, 0)
	}

	rs := bytes.NewReader(b)
	md, err := tag.ReadFrom(rs)
	if err != nil {
		return res, errors.Wrap(err, 0)
	}

	// Package tag provides these already, but let's set them explicitly
	// here just in case the constants change with an upgrade, since they
	// can have a functional impact.
	var fileType string
	switch md.FileType() {
	case tag.FLAC:
		fileType = "FLAC"
		res.ContentType = "audio/flac"
	case tag.OGG:
		fileType = "Ogg"
		res.ContentType = "audio/ogg"
	case tag.MP3:
		fileType = "MP3"
		res.ContentType = "audio/mpeg"
	default:
		return res, fmt.Errorf("%w: unsupported music file format %q", ErrSkipped, md.FileType())
	}

	track, trackTotal := md.Track()
	disc, discTotal := md.Disc()

	res.Metadata = TrackInfo{
		FileType:    fileType,
		Title:       md.Title(),
		Album:       md.Album(),
		Artist:      md.Artist(),
		AlbumArtist: md.AlbumArtist(),
		Composer:    md.Composer(),
		Genre:       md.Genre(),
		Year:        md.Year(),
		Track:       track,
		TrackTotal:  trackTotal,
		Disc:        disc,
		DiscTotal:   discTotal,
		Lyrics:      md.Lyrics(),
		Comment:     md.Comment(),
	}

	return res, nil
}

type TrackInfo struct {
	FileType string

	// Below are provided directly by the tag package.

	Title       string
	Album       string
	Artist      string
	AlbumArtist string
	Composer    string
	Genre       string
	Year        int
	Track       int
	TrackTotal  int
	Disc        int
	DiscTotal   int
	Lyrics      string
	Comment     string
}
