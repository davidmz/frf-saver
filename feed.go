package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
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

type WithRawBody struct {
	RawBody string `json:"rawBody"`
}

type FromEntry struct {
	From struct {
		ID string `json:"id"`
	} `json:"from"`
}

func (e *Entry) JustID() string { return e.Id[2:] }

func (s *Saver) loadFeed(apiReq, destDir string) (eerr error) {
	s.Log.INFO("Reading feed %v", apiReq)
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
		URL := ApiRoot + apiReq

		if strings.Contains(URL, "?") {
			URL += "&"
		} else {
			URL += "?"
		}
		URL += "num=100&hidden=1&raw=1&start=" + strconv.Itoa(start)

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
			s.Log.INFO("The end")
			break
		} else {
			lastId = feed.Entries[0].Id
		}

		for _, e := range feed.Entries {
			f, _ := os.Create(filepath.Join(s.OutDirName, s.FeedId, destDir, e.JustID()+".json"))
			json.NewEncoder(f).Encode(e)
			f.Close()

			s.processEntry(e)
		}
		start += len(feed.Entries)
	}
	return
}

func (s *Saver) processEntry(e *Entry) {
	s.loadLinks(e.RawBody)

	// media
	for _, m := range e.Files {
		s.loadMedia(m)
	}
	for _, m := range e.Thumbnails {
		s.loadMedia(m)
	}

	// avatars
	fr := new(FromEntry)
	for _, m := range e.Comments {
		json.Unmarshal(m, fr)
		s.loadAvatar(fr.From.ID)
	}

	for _, m := range e.Likes {
		json.Unmarshal(m, fr)
		s.loadAvatar(fr.From.ID)
	}

	// comment's bodies
	b := new(WithRawBody)
	for _, m := range e.Comments {
		json.Unmarshal(m, b)
		s.loadLinks(b.RawBody)
	}
}

var ffImRe = regexp.MustCompile(`http://ff.im/\w+`)

func (s *Saver) loadLinks(body string) {
	for _, u := range ffImRe.FindAllString(body, -1) {
		s.Log.DEBUG("Found link %v", u)
		uu, _ := url.Parse(u)

		req, _ := http.NewRequest("GET", ApiRoot+"short"+uu.Path, nil)
		req.SetBasicAuth(s.Username, s.RemoteKey)
		resp, _ := http.DefaultClient.Do(req)

		fileName := filepath.Join(s.OutDirName, s.FeedId, "links", filepath.FromSlash(uu.Host+uu.Path)+".json")
		tmpFileName := filepath.Join(os.TempDir(), "frf-saver-link-"+url.QueryEscape(uu.Host+uu.Path))

		os.MkdirAll(filepath.Dir(fileName), os.ModePerm)

		e := new(Entry)

		f, _ := os.Create(tmpFileName)

		tr := io.TeeReader(resp.Body, f)
		json.NewDecoder(tr).Decode(e)

		f.Close()
		resp.Body.Close()
		os.Rename(tmpFileName, fileName)

		s.processEntry(e)
	}
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

		tmpFileName := filepath.Join(os.TempDir(), "frf-saver-media-"+url.QueryEscape(uu.Host+uu.Path))
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
			case "audio/mpeg", "audio/mp3":
				fileName += ".mp3"
			default:
				s.Log.DEBUG("Unknown content type: %s", resp.Header.Get("Content-Type"))
			}
		}

		f, _ := os.Create(tmpFileName)
		io.Copy(f, resp.Body)
		f.Close()
		resp.Body.Close()

		os.Rename(tmpFileName, fileName)
	}
}
