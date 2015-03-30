package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Feed struct {
	Entries []*Entry `json:"entries`
}

type Entry struct {
	Id         string            `json:"id"`
	URL        string            `json:"url"`
	Date       string            `json:"date"`
	Body       string            `json:"body"`
	RawBody    string            `json:"rawBody"`
	From       json.RawMessage   `json:"from"`
	To         json.RawMessage   `json:"to,omitempty"`
	Comments   []json.RawMessage `json:"comments,omitempty"`
	Likes      []json.RawMessage `json:"likes,omitempty"`
	Thumbnails []json.RawMessage `json:"thumbnails,omitempty"`
	Files      []json.RawMessage `json:"files,omitempty"`
	Via        json.RawMessage   `json:"via,omitempty"`
	Geo        json.RawMessage   `json:"geo,omitempty"`
	Commands   json.RawMessage   `json:"commands,omitempty"`
}

type Media struct {
	Url  string `json:"url"`
	Link string `json:"link"`
	Icon string `json:"icon"`
}

type FromEntry struct {
	From struct {
		ID string `json:"id"`
	} `json:"from"`
}

func (e *Entry) JustID() string { return e.Id[2:] }

func (s *Saver) loadFeed() (eerr error) {
	s.Log.INFO("Reading feed")
	defer func() {
		if eerr != nil {
			s.Log.FATAL("Can not read feed: %v", eerr)
		}
	}()

	eerr = os.MkdirAll(filepath.Join(s.OutDirName, s.FeedId, "entries"), os.ModePerm)
	if eerr != nil {
		return
	}

	start := 0
	lastId := ""
	for {
		s.Log.INFO("Loading from %d", start)
		URL := ApiRoot + "feed/" + s.FeedId + "?num=100&hidden=1&raw=1&start=" + strconv.Itoa(start)
		req, _ := http.NewRequest("GET", URL, nil)
		req.SetBasicAuth(s.Username, s.RemoteKey)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		feed := new(Feed)
		eerr = json.NewDecoder(resp.Body).Decode(feed)
		if eerr != nil {
			return
		}

		if len(feed.Entries) == 0 || feed.Entries[0].Id == lastId {
			s.Log.INFO("The end.")
			break
		}

		lastId = feed.Entries[0].Id

		for _, e := range feed.Entries {
			f, _ := os.Create(filepath.Join(s.OutDirName, s.FeedId, "entries", e.JustID()+".json"))
			json.NewEncoder(f).Encode(e)
			f.Close()

			// media
			for _, m := range e.Files {
				s.loadMedia(m)
			}
			for _, m := range e.Thumbnails {
				s.loadMedia(m)
			}

			// avatars
			for _, m := range e.Comments {
				fr := new(FromEntry)
				json.Unmarshal(m, fr)
				s.loadAvatar(fr.From.ID)
			}
			for _, m := range e.Likes {
				fr := new(FromEntry)
				json.Unmarshal(m, fr)
				s.loadAvatar(fr.From.ID)
			}
		}
		start += 100
	}
	return
}

func (s *Saver) loadMedia(rm json.RawMessage) {
	m := new(Media)
	json.Unmarshal(rm, m)
	s.Async(func() { s.loadUrl(m.Link) })
	s.Async(func() { s.loadUrl(m.Url) })
	s.Async(func() { s.loadUrl(m.Icon) })
}

func (s *Saver) loadUrl(u string) {
	if u == "" {
		return
	}
	uu, err := url.Parse(u)
	if err != nil {
		return
	}

	if uu.Host == "m.friendfeed-media.com" ||
		uu.Host == "i.friendfeed.com" ||
		uu.Host == "friendfeed.com" && strings.HasPrefix(uu.Path, "/static/") ||
		// uu.Host == "friendfeed-media.s3.amazonaws.com" || // always same as "m.friendfeed-media.com"
		false {
		s.Log.TRACE("loading %s", u)

		fileName := filepath.Join(s.OutDirName, s.FeedId, "media", filepath.FromSlash(uu.Host+uu.Path))

		glb := fileName
		if filepath.Ext(fileName) == "" {
			glb += ".*"
		}

		if fs, _ := filepath.Glob(glb); len(fs) > 0 {
			s.Log.TRACE("already loaded %s", u)
			return
		}

		os.MkdirAll(filepath.Dir(fileName), os.ModePerm)

		resp, err := http.Get(u)
		if err != nil {
			s.Log.ERROR("error loading %s: %v", u, err)
			return
		}

		if filepath.Ext(fileName) == "" {
			switch resp.Header.Get("Content-Type") {
			case "image/jpeg":
				fileName += ".jpg"
			case "image/png":
				fileName += ".png"
			case "image/gif":
				fileName += ".gif"
			case "audio/mpeg":
				fileName += ".mp3"
			default:
				s.Log.DEBUG("Unknown content type: %s", resp.Header.Get("Content-Type"))
			}
		}

		f, _ := os.Create(fileName)
		io.Copy(f, resp.Body)
		f.Close()
		resp.Body.Close()
	}
}
