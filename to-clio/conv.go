package main

import (
	"encoding/json"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/davidmz/logg"
)

type Converter struct {
	SrcDir string
	TgtDir string
	Log    *logg.Logger

	Images map[string]string
	Files  map[string]string
}

type Entry struct {
	Id         string            `json:"id"`
	URL        string            `json:"url"`
	Date       string            `json:"date"`
	Body       string            `json:"body"`
	RawBody    string            `json:"rawBody,omitempty"`
	From       json.RawMessage   `json:"from"`
	To         json.RawMessage   `json:"to,omitempty"`
	Comments   []*Comment        `json:"comments"`
	Likes      []json.RawMessage `json:"likes"`
	Thumbnails []json.RawMessage `json:"thumbnails,omitempty"`
	Files      []json.RawMessage `json:"files,omitempty"`
	Via        json.RawMessage   `json:"via,omitempty"`
	Geo        json.RawMessage   `json:"geo,omitempty"`
	Commands   json.RawMessage   `json:"commands,omitempty"`

	Name string `json:"name,omitempty"`
	// Russian.strftime(parse_time(t), '%d %B %Y в %H:%M')
	DateFriendly string `json:"dateFriendly,omitempty"`
}

type Comment struct {
	Id       string          `json:"id"`
	Date     string          `json:"date"`
	Body     string          `json:"body"`
	RawBody  string          `json:"rawBody,omitempty"`
	From     json.RawMessage `json:"from"`
	Via      json.RawMessage `json:"via,omitempty"`
	Commands json.RawMessage `json:"commands,omitempty"`

	DateFriendly string `json:"dateFriendly,omitempty"`
}

type Media struct {
	Url  string `json:"url"`
	Link string `json:"link"`
}

func (c *Converter) Convert() {
	c.Images = make(map[string]string)
	c.Files = make(map[string]string)

	userpicsDir := filepath.Join(c.TgtDir, "images", "userpics")
	jsonDataDir := filepath.Join(c.TgtDir, "_json", "data")
	jsonEntriesDir := filepath.Join(jsonDataDir, "entries")
	if err := os.MkdirAll(userpicsDir, os.ModePerm); err != nil {
		c.Log.ERROR("Can not create dir: %v", err)
		return
	}
	os.MkdirAll(jsonDataDir, os.ModePerm)
	os.MkdirAll(jsonEntriesDir, os.ModePerm)
	os.MkdirAll(filepath.Join(c.TgtDir, "files"), os.ModePerm)
	os.MkdirAll(filepath.Join(c.TgtDir, "images", "media", "thumbnails"), os.ModePerm)

	if err := CopyFile(
		filepath.Join(c.SrcDir, "feedinfo.json"),
		filepath.Join(jsonDataDir, "feedinfo.js"),
	); err != nil {
		c.Log.ERROR("Can not copy feedinfo dir: %v", err)
		return
	}

	c.Log.INFO("Converting entries")
	eFiles, _ := filepath.Glob(filepath.Join(c.SrcDir, "entries", "*.json"))
	for _, eFile := range eFiles {
		c.ConvertEntry(eFile, jsonEntriesDir)
	}

	c.Log.INFO("Writing images.tsv")
	f, _ := os.Create(filepath.Join(jsonDataDir, "images.tsv"))
	for k, v := range c.Images {
		f.WriteString(k + "\t" + v + "\n")
	}
	f.Close()

	c.Log.INFO("Writing files.tsv")
	f, _ = os.Create(filepath.Join(jsonDataDir, "files.tsv"))
	for k, v := range c.Files {
		f.WriteString(k + "\t" + v + "\n")
	}
	f.Close()

	c.Log.INFO("Copying userpics")
	uFiles, _ := filepath.Glob(filepath.Join(c.SrcDir, "avatars", "*.jpg"))
	for _, uFile := range uFiles {
		CopyFile(uFile, filepath.Join(userpicsDir, filepath.Base(uFile)))
	}

	c.Log.INFO("All done")
}

var monReplacer = strings.NewReplacer(
	"Month", "января",
	"February", "февраля",
	"March", "марта",
	"April", "апреля",
	"May", "мая",
	"June", "июня",
	"July", "июля",
	"August", "августа",
	"September", "сентября",
	"October", "октября",
	"November", "ноября",
	"December", "декабря",
)

func (c *Converter) ConvertEntry(fName, tgtDir string) {
	e := new(Entry)
	f, _ := os.Open(fName)
	defer f.Close()

	if err := json.NewDecoder(f).Decode(e); err != nil {
		c.Log.ERROR("Can not decode entry %q", fName)
		return
	}

	e.Name = e.Id[2:10]
	t, _ := time.Parse(time.RFC3339, e.Date)
	e.DateFriendly = monReplacer.Replace(t.Format("02 January 2006 в 15:04"))
	e.RawBody = ""

	for _, comm := range e.Comments {
		t, _ := time.Parse(time.RFC3339, comm.Date)
		comm.DateFriendly = monReplacer.Replace(t.Format("02 January 2006 в 15:04"))
		comm.RawBody = ""
	}

	out, _ := os.Create(filepath.Join(tgtDir, e.Name+".js"))
	json.NewEncoder(out).Encode(e)
	out.Close()

	for _, rm := range e.Thumbnails {
		m := new(Media)
		json.Unmarshal(rm, m)

		uu, _ := url.Parse(m.Url)
		if uu.Host == "m.friendfeed-media.com" || uu.Host == "i.friendfeed.com" {
			srcName := filepath.Join(c.SrcDir, "media", filepath.FromSlash(uu.Host+uu.Path)+".jpg")
			CopyFile(srcName, filepath.Join(c.TgtDir, "images", "media", "thumbnails", filepath.Base(srcName)))
		}

		uu, _ = url.Parse(m.Link)
		if uu.Host == "m.friendfeed-media.com" || uu.Host == "i.friendfeed.com" {
			glb := filepath.Join(c.SrcDir, "media", filepath.FromSlash(uu.Host+uu.Path)+".*")
			if fs, _ := filepath.Glob(glb); len(fs) > 0 {
				CopyFile(fs[0], filepath.Join(c.TgtDir, "images", "media", filepath.Base(fs[0])))
				c.Images[uu.String()] = filepath.Base(fs[0])
			}
		}
	}

	for _, rm := range e.Files {
		m := new(Media)
		json.Unmarshal(rm, m)

		uu, _ := url.Parse(m.Url)
		if uu.Host == "m.friendfeed-media.com" || uu.Host == "i.friendfeed.com" {
			glb := filepath.Join(c.SrcDir, "media", filepath.FromSlash(uu.Host+uu.Path)+".*")
			if fs, _ := filepath.Glob(glb); len(fs) > 0 {
				CopyFile(fs[0], filepath.Join(c.TgtDir, "files", filepath.Base(fs[0])))
				c.Files[uu.String()] = filepath.Base(fs[0])
			}
		}
	}

}

func CopyFile(src, tgt string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(tgt)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
