package main

import (
	"io"
	"os"
	"path/filepath"
)

var (
	avatarsInQueue = make(map[string]struct{})
	nope           = struct{}{}
)

func (s *Saver) loadAvatar(login string) {
	if _, ok := avatarsInQueue[login]; ok {
		// already exists
		s.Log.DEBUG("Avatar already loaded %q", login)
		return
	}
	avatarsInQueue[login] = nope

	if _, err := os.Stat(filepath.Join(s.BaseDirName(), "avatars", login+".jpg")); err == nil {
		// local file exists
		s.Log.DEBUG("Local avatar exists %q", login)
		return
	}

	s.Async(func() { s.loadAvatarData(login) })
}

func (s *Saver) loadAvatarData(login string) {
	u := ApiRoot + "picture/" + login + "?size=large"
	resp, err := s.GetReq(u)
	if err != nil {
		s.Log.ERROR("error loading %s: %v", u, err)
		return
	}

	fileName := filepath.Join(s.BaseDirName(), "avatars", login+".jpg")
	tmpFileName := fileName + ".tmp"

	f, _ := os.Create(tmpFileName)
	io.Copy(f, resp.Body)
	f.Close()
	resp.Body.Close()

	os.Rename(tmpFileName, fileName)
}
