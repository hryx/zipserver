package zipserver

import (
	"bytes"
	"fmt"
	"io"

	"github.com/dhowden/tag"
)

// MusicAnalyzer uses rules according to music albums.
// FLAC, Ogg, and MP3 files are supported, and all other files are ignored.
// Song metadata is collected and returned.
type MusicAnalyzer struct{}

func (m MusicAnalyzer) Analyze(r io.Reader, key string) (AnalyzeResult, error) {
	res := AnalyzeResult{}

	// TODO: The music tag library requires a ReadSeeker (understandably),
	// which a Reader does not satisfy. Here is a naive implementation that
	// reads the entire file into memory. To lower memory usage, this should
	// be replaced with a custom ReadSeeker that uses a limited buffer for
	// the unfortunately necessary seeks.

	b, err := io.ReadAll(r)
	if err != nil {
		return res, fmt.Errorf("read bytes: %w", err)
	}

	rs := bytes.NewReader(b)
	md, err := tag.ReadFrom(rs)
	if err != nil {
		// Tag parser expects at least 11 bytes.
		if err == io.ErrUnexpectedEOF {
			return res, fmt.Errorf("%w: file too short", ErrSkipped)
		}
		return res, fmt.Errorf("new tag reader: %w", err)
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
	res.Key = key

	return res, nil
}

type TrackInfo struct {
	FileType string

	// Below are provided directly by the tag package.

	Title       string `json:",omitempty"`
	Album       string `json:",omitempty"`
	Artist      string `json:",omitempty"`
	AlbumArtist string `json:",omitempty"`
	Composer    string `json:",omitempty"`
	Genre       string `json:",omitempty"`
	Year        int    `json:",omitempty"`
	Track       int    `json:",omitempty"`
	TrackTotal  int    `json:",omitempty"`
	Disc        int    `json:",omitempty"`
	DiscTotal   int    `json:",omitempty"`
	Lyrics      string `json:",omitempty"`
	Comment     string `json:",omitempty"`
}
