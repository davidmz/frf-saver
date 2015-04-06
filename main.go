package main

import (
	"flag"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/davidmz/go-semaphore"
	"github.com/davidmz/logg"
)

const (
	ApiRoot = "http://friendfeed-api.com/v2/"
)

type Conf struct {
	Username   string
	RemoteKey  string
	OutDirName string
	AllFriends bool
	Log        *logg.Logger
	WG         *sync.WaitGroup
	Sem        *semaphore.Semaphore
	NRetries   int
	JustMedia  bool
	SaveLikes  bool
	AuthCookie string
}

type Saver struct {
	*Conf
	FeedId  string
	Friends []string
	Log     *logg.Logger
}

func main() {
	conf := new(Conf)

	conf.WG = new(sync.WaitGroup)
	conf.Sem = semaphore.New(50)

	logLevel := ""
	feedId := ""
	nWorkers := 1

	flag.StringVar(&conf.Username, "u", "", "username to login")
	flag.StringVar(&feedId, "f", "", "feed name to load (your username if not setted)")
	flag.BoolVar(&conf.AllFriends, "a", false, "save 'username' and all his/her subscriptions (-f ignored)")
	flag.StringVar(&conf.RemoteKey, "k", "", "remote key (see https://friendfeed.com/account/api)")
	flag.StringVar(&conf.OutDirName, "d", "./frf-save", "directory to save data")
	flag.StringVar(&logLevel, "ll", "info", "log level")
	flag.IntVar(&conf.NRetries, "r", 5, "number of network retries")
	flag.IntVar(&nWorkers, "w", 1, "number of parallel workers for '-a'")
	flag.BoolVar(&conf.JustMedia, "m", false, "just check and load missing media files for loaded entries")
	flag.BoolVar(&conf.SaveLikes, "save-likes", false, "save all likes in depth (DESTRUCTIVE, only owner's likes)")
	flag.StringVar(&conf.AuthCookie, "coo", "", "value of frf cookie 'U' (for -save-likes)")
	flag.Parse()

	if conf.Username == "" || conf.RemoteKey == "" {
		flag.Usage()
		os.Exit(1)
	}

	if conf.SaveLikes {
		feedId = ""
		conf.AllFriends = false
	}

	if feedId == "" {
		feedId = conf.Username
	}

	ll, err := logg.LevelByName(logLevel)
	if err != nil {
		ll = logg.WARN
	}

	if nWorkers < 1 {
		nWorkers = 1
	}

	conf.Log = logg.New(ll, logg.DefaultWriter)

	loadQ := make(chan string)

	if conf.AllFriends && feedId == conf.Username {
		saver := &Saver{Conf: conf, FeedId: feedId}
		conf.Log.INFO("Loading friends of %q", feedId)
		saver.loadFeedInfo()
		go func(friends []string) {
			sort.Strings(friends)
			for _, f := range friends {
				loadQ <- f
			}
			close(loadQ)
		}(append(saver.Friends, saver.FeedId))

		wwg := new(sync.WaitGroup)
		for i := 0; i < nWorkers; i++ {
			wwg.Add(1)
			go func() {
				defer wwg.Done()
				for {
					feedId, ok := <-loadQ
					if !ok {
						break
					}
					saver := &Saver{Conf: conf, FeedId: feedId}
					saver.process()
				}
			}()
		}
		wwg.Wait()

	} else {
		saver := &Saver{Conf: conf, FeedId: feedId}
		saver.process()
	}

	go func() {
		for range time.Tick(2 * time.Second) {
			conf.Log.INFO("%d file(s) still downloads, please wait", conf.Sem.AcquiredCount()+conf.Sem.WaitingCount())
		}
	}()

	conf.WG.Wait()

	conf.Log.INFO("All done")
}

func (saver *Saver) process() {
	saver.Log = saver.Conf.Log.ChildWithPrefix(saver.FeedId)

	if saver.SaveLikes {
		saver.saveLikes()
		return
	}

	if saver.JustMedia {
		saver.processMedia()
		return
	}

	os.MkdirAll(filepath.Join(saver.BaseDirName(), "entries"), os.ModePerm)
	os.MkdirAll(filepath.Join(saver.BaseDirName(), "liked"), os.ModePerm)
	os.MkdirAll(filepath.Join(saver.BaseDirName(), "commented"), os.ModePerm)
	os.MkdirAll(filepath.Join(saver.BaseDirName(), "media"), os.ModePerm)
	os.MkdirAll(filepath.Join(saver.BaseDirName(), "avatars"), os.ModePerm)

	saver.Log.Prefix = saver.FeedId + ":info"
	saver.loadFeedInfo()

	saver.Log.Prefix = saver.FeedId + ":likes"
	saver.loadFeed("search?q=like:"+saver.FeedId, "liked")

	saver.Log.Prefix = saver.FeedId + ":comms"
	saver.loadFeed("search?q=comment:"+saver.FeedId, "commented")

	saver.Log.Prefix = saver.FeedId + ":feed"
	saver.loadFeed("feed/"+saver.FeedId, "entries")
}

func (s *Saver) BaseDirName() string { return filepath.Join(s.OutDirName, s.FeedId) }

func (s *Saver) Async(foo func()) {
	s.WG.Add(1)
	go func() {
		defer s.Sem.Acquire().Release()
		defer s.WG.Done()
		foo()
	}()
}

func (c *Conf) DoReq(req *http.Request) (resp *http.Response, err error) {
	for i := 0; i < c.NRetries; i++ {
		resp, err = http.DefaultClient.Do(req)
		if err == nil {
			break
		}
	}
	return
}

func (c *Conf) GetReq(u string) (*http.Response, error) {
	req, _ := http.NewRequest("GET", u, nil)
	return c.DoReq(req)
}
