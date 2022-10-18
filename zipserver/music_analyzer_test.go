package zipserver

import (
	"bytes"
	"embed"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

//go:embed testdata/music/*
var testAudioFiles embed.FS

func TestMusicAnalyzer(t *testing.T) {
	testCases := []struct {
		name string
		err  error
		res  AnalyzeResult
	}{
		{
			name: "tone.thisisflac",
			res: AnalyzeResult{
				Key:         "key",
				ContentType: "audio/flac",
				Metadata: TrackInfo{
					FileType:    "FLAC",
					Title:       "Long drone",
					Album:       "Fall asleep",
					Artist:      "Smith",
					AlbumArtist: "Smith",
					Composer:    "Smith",
					Year:        2005,
					Track:       4,
					TrackTotal:  6,
					Disc:        2,
					DiscTotal:   2,
				},
			},
		},
		{
			name: "tone.thisisopus",
			res: AnalyzeResult{
				Key:         "key",
				ContentType: "audio/ogg",
				Metadata: TrackInfo{
					FileType:   "Ogg",
					Title:      "Halloween song 2",
					Album:      "Don't listen to any of these songs",
					Artist:     "Creepy Weirdo",
					Genre:      "horror",
					Year:       2022,
					Track:      12,
					TrackTotal: 666,
					Lyrics:     "pumpkins yeah\\ndecorate your house",
					Comment:    "spooky tune",
				},
			},
		},
		{
			name: "tone.thisismp3",
			res: AnalyzeResult{
				Key:         "key",
				ContentType: "audio/mpeg",
				Metadata: TrackInfo{
					FileType:   "MP3",
					Title:      "Boring",
					Album:      "Wat?",
					Artist:     "Small Dude",
					Genre:      "Acoustic",
					Year:       1994,
					Track:      2,
					TrackTotal: 10,
					Comment:    "wrote this when I was bored",
				},
			},
		},
		{
			name: "garbage.dat",
			err:  ErrSkipped,
		},
	}

	var analyzer MusicAnalyzer

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b, err := testAudioFiles.ReadFile("testdata/music/" + tc.name)
			require.NoError(t, err)
			buf := bytes.NewBuffer(b)
			res, err := analyzer.Analyze(buf, "key")
			require.Truef(t, errors.Is(err, tc.err), "error %q does not wrap %q", err, tc.err)
			assert.Equal(t, tc.res, res)
		})
	}
}
