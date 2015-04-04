package main

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FInfo struct {
	Subscribers []*struct {
		ID string `json:"id"`
	} `json:"subscribers"`
	Subscriptions []*struct {
		ID string `json:"id"`
	} `json:"subscriptions"`
}

func (s *Saver) loadFeedInfo() (eerr error) {
	s.Log.INFO("Reading feed info")
	defer func() {
		if eerr != nil {
			s.Log.FATAL("Can not read feed info: %v", eerr)
		}
	}()

	eerr = os.MkdirAll(filepath.Join(s.OutDirName, s.FeedId), os.ModePerm)
	if eerr != nil {
		return
	}

	outFile, eerr := os.Create(filepath.Join(s.OutDirName, s.FeedId, "feedinfo.json"))
	if eerr != nil {
		return
	}
	defer outFile.Close()

	req, _ := http.NewRequest("GET", ApiRoot+"feedinfo/"+s.FeedId+"?pretty=1", nil)
	req.SetBasicAuth(s.Username, s.RemoteKey)
	resp, eerr := s.DoReq(req)
	if eerr != nil {
		return
	}
	defer resp.Body.Close()

	tr := io.TeeReader(resp.Body, outFile)

	fi := new(FInfo)
	json.NewDecoder(tr).Decode(fi)

	s.Friends = make([]string, len(fi.Subscriptions))
	for i, sub := range fi.Subscriptions {
		s.Friends[i] = sub.ID
		s.loadAvatar(sub.ID)
	}
	for _, sub := range fi.Subscribers {
		s.loadAvatar(sub.ID)
	}

	return
}
