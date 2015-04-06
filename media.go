package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

func (s *Saver) processMedia() {
	s.Log.INFO("Loading media")
	entriesDirs := []string{
		filepath.Join(s.BaseDirName(), "entries"),
		filepath.Join(s.BaseDirName(), "liked"),
		filepath.Join(s.BaseDirName(), "commented"),
	}

	for _, dir := range entriesDirs {
		eFiles, _ := filepath.Glob(filepath.Join(dir, "*.json"))
		s.Log.DEBUG("%s: found %d entries", filepath.Base(dir), len(eFiles))
		for _, fName := range eFiles {
			e := new(Entry)
			f, _ := os.Open(fName)
			json.NewDecoder(f).Decode(e)
			f.Close()

			for _, m := range e.Files {
				s.loadMedia(m)
			}
			for _, m := range e.Thumbnails {
				s.loadMedia(m)
			}

			s.loadLinks(e.RawBody)

			// comment's bodies
			b := new(WithRawBody)
			fr := new(FromEntry)
			for _, m := range e.Comments {
				json.Unmarshal(m, b)
				json.Unmarshal(m, fr)
				s.loadLinks(b.RawBody)
				s.loadAvatar(fr.From.ID)
			}
		}
	}
}
