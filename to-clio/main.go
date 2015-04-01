package main

import (
	"os"
	"path/filepath"
)
import (
	"github.com/davidmz/logg"
	"github.com/docopt/docopt-go"
)

func main() {
	usage := `to-clio
Converter from frf-saver to clio format.

Usage:
  to-clio [options] FROM_DIR TO_DIR
  to-clio -h | --help

Options:
  -h, --help  Show this help and exit
  -ll LEVEL   Log level [default: info]

FROM_DIR - directory with frf-saver backup (individual feed or all feeds)
TO_DIR   - target directory ('result' in Clio)
`
	arguments, _ := docopt.Parse(usage, nil, true, "to-clio", false)

	logLevel, err := logg.LevelByName(arguments["-ll"].(string))
	if err != nil {
		logLevel = logg.INFO
	}

	log := logg.New(logLevel, logg.DefaultWriter)

	fromDir, err := filepath.Abs(arguments["FROM_DIR"].(string))
	if err != nil {
		log.FATAL("Invalid path: %v", err)
		os.Exit(1)
	}
	toDir, err := filepath.Abs(arguments["TO_DIR"].(string))
	if err != nil {
		log.FATAL("Invalid path: %v", err)
		os.Exit(1)
	}
	if _, err := os.Stat(fromDir); err != nil {
		log.FATAL("Directory %q not exists", fromDir)
		os.Exit(1)
	}

	if _, err := os.Stat(filepath.Join(fromDir, "feedinfo.json")); err == nil {

		conv := &Converter{
			SrcDir: fromDir,
			TgtDir: filepath.Join(toDir, filepath.Base(fromDir)),
			Log:    log,
		}
		log.INFO("%v → %v", conv.SrcDir, conv.TgtDir)

		conv.Convert()

	} else {
		// Предполагаем, что профили хранятся в каталогах первого уровня
		dir, _ := os.Open(fromDir)
		subdirs, _ := dir.Readdirnames(-1)
		dir.Close()

		for _, name := range subdirs {
			fd := filepath.Join(fromDir, name)
			if _, err := os.Stat(filepath.Join(fd, "feedinfo.json")); err == nil {

				conv := &Converter{
					SrcDir: fd,
					TgtDir: filepath.Join(toDir, filepath.Base(fd)),
					Log:    log,
				}
				log.INFO("%v → %v", conv.SrcDir, conv.TgtDir)

				conv.Convert()
			}
		}

	}

}
