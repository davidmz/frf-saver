package main

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/davidmz/go-semaphore"
)

type AtomFeed struct {
	Ids []string `xml:"entry>id"`
}

func (s *Saver) saveLikes() {
	s.Log.INFO("Fetching likes...")

	destDir := "liked"

	lastId := ""
	nDislikes := 0
	for {
		s.Log.INFO("Loading first 100 likes (and dislike its)")

		var eIds []string
		{
			uu := "http://friendfeed.com/" + s.FeedId + "/likes?num=100&format=atom"
			req, _ := http.NewRequest("GET", uu, nil)
			req.AddCookie(&http.Cookie{Name: "U", Value: s.AuthCookie})
			resp, err := s.DoReq(req)
			if err != nil {
				s.Log.ERROR("Can not read feed: %v", err)
				return
			}
			af := new(AtomFeed)
			xml.NewDecoder(resp.Body).Decode(af)
			resp.Body.Close()
			eIds = make([]string, len(af.Ids))
			for i, sid := range af.Ids {
				idx := strings.Index(sid, "2007:")
				eIds[i] = "e/" + strings.Replace(sid[idx+len("2007:"):], "-", "", -1)
			}
		}

		req, _ := http.NewRequest("GET", ApiRoot+"entry?raw=1&hidden=1&id="+strings.Join(eIds, ","), nil)
		req.SetBasicAuth(s.Username, s.RemoteKey)
		resp, err := s.DoReq(req)
		if err != nil {
			s.Log.FATAL("Can not load feed: %v", err)
			return
		}

		feed := new(Feed)
		if err := json.NewDecoder(resp.Body).Decode(feed); err != nil {
			s.Log.FATAL("Can not parse XML: %v", err)
			resp.Body.Close()
			return
		}
		resp.Body.Close()

		if len(feed.Entries) == 0 {
			s.Log.INFO("The end")
			break
		} else if feed.Entries[0].Id == lastId {
			s.Log.INFO("Same result, waiting")
			time.Sleep(10 * time.Second)
			continue
		} else {
			lastId = feed.Entries[0].Id
		}

		wg := new(sync.WaitGroup)
		sem := semaphore.New(3)

		for _, e := range feed.Entries {
			f, err := os.Create(filepath.Join(s.OutDirName, s.FeedId, destDir, e.JustID()+".json"))
			if err != nil {
				s.Log.ERROR("Can not write entry: %v", err)
				return
			}
			if err := json.NewEncoder(f).Encode(e); err != nil {
				s.Log.ERROR("Can not encode entry: %v", err)
				return
			}
			f.Close()
			// разлайкиваем
			wg.Add(1)
			go func(eId string) {
				defer wg.Done()
				defer sem.Acquire().Release()

				req, _ := http.NewRequest("POST", ApiRoot+"like/delete", strings.NewReader("entry="+eId))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.SetBasicAuth(s.Username, s.RemoteKey)
				resp, err := s.DoReq(req)
				if err != nil {
					s.Log.ERROR("Can not dislike: %v", err)
					return
				}
				//io.Copy(ioutil.Discard, resp.Body)
				b, _ := ioutil.ReadAll(resp.Body)
				s.Log.TRACE("dislike response: %s %s", string(b), eId)
				resp.Body.Close()
				nDislikes++
				if nDislikes%10 == 0 {
					s.Log.INFO("%d entries disliked", nDislikes)
				}
			}(e.Id)
		}
		wg.Wait()
	}
	s.Log.INFO("%d entries disliked", nDislikes)
}
