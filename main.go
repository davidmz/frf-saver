package main

import (
	"flag"
	"os"
	"path/filepath"
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
}

type Saver struct {
	*Conf
	FeedId  string
	Friends []string
}

func main() {
	conf := new(Conf)
	saver := &Saver{Conf: conf}

	saver.WG = new(sync.WaitGroup)
	saver.Sem = semaphore.New(50)

	logLevel := ""

	flag.StringVar(&saver.Username, "u", "", "username to login")
	flag.StringVar(&saver.FeedId, "f", "", "feed name to load (your username if not setted)")
	flag.BoolVar(&saver.AllFriends, "a", false, "save 'username' and all his/her subscriptions (-f ignored)")
	flag.StringVar(&saver.RemoteKey, "k", "", "remote key (see https://friendfeed.com/account/api)")
	flag.StringVar(&saver.OutDirName, "d", "./frf-save", "directory to save data")
	flag.StringVar(&logLevel, "ll", "info", "log level")
	flag.Parse()

	if saver.Username == "" || saver.RemoteKey == "" {
		flag.Usage()
		os.Exit(1)
	}

	if saver.FeedId == "" {
		saver.FeedId = saver.Username
	}

	ll, err := logg.LevelByName(logLevel)
	if err != nil {
		ll = logg.WARN
	}

	saver.Log = logg.New(ll, logg.DefaultWriter)
	saver.Log.Prefix = saver.FeedId

	var toLoad []string
	if conf.AllFriends {
		toLoad = []string{conf.Username}
	} else {
		toLoad = []string{saver.FeedId}
	}

	for len(toLoad) != 0 {
		saver = &Saver{Conf: conf, FeedId: toLoad[0]}
		saver.Log.Prefix = saver.FeedId
		saver.Log.INFO("Loading %q", saver.FeedId)

		toLoad = toLoad[1:]

		os.MkdirAll(filepath.Join(saver.BaseDirName(), "entries"), os.ModePerm)
		os.MkdirAll(filepath.Join(saver.BaseDirName(), "liked"), os.ModePerm)
		os.MkdirAll(filepath.Join(saver.BaseDirName(), "commented"), os.ModePerm)
		os.MkdirAll(filepath.Join(saver.BaseDirName(), "media"), os.ModePerm)
		os.MkdirAll(filepath.Join(saver.BaseDirName(), "avatars"), os.ModePerm)

		saver.loadFeedInfo()

		if saver.FeedId == saver.Username {
			saver.loadFeed("search?q=like:"+saver.FeedId, "liked")
			saver.loadFeed("search?q=comment:"+saver.FeedId, "commented")
		}

		saver.loadFeed("feed/"+saver.FeedId, "entries")

		if conf.AllFriends && saver.FeedId == conf.Username {
			toLoad = saver.Friends
		}
	}

	go func() {
		for range time.Tick(2 * time.Second) {
			saver.Log.INFO("%d file(s) still downloads, please wait", conf.Sem.AcquiredCount()+conf.Sem.WaitingCount())
		}
	}()

	saver.WG.Wait()

	saver.Log.INFO("All done")
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
