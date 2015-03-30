package main

import (
	"io"
	"net/http"
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
		return
	}
	avatarsInQueue[login] = nope

	s.Async(func() { s.loadAvatarData(login) })
}

func (s *Saver) loadAvatarData(login string) {
	u := ApiRoot + "picture/" + login + "?size=large"
	resp, err := http.Get(u)
	if err != nil {
		s.Log.ERROR("error loading %s: %v", u, err)
		return
	}

	f, _ := os.Create(filepath.Join(s.OutDirName, s.FeedId, "avatars", login+".jpg"))
	io.Copy(f, resp.Body)
	f.Close()
	resp.Body.Close()
}
