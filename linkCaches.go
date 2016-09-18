package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/syndtr/goleveldb/leveldb"
)

var parserDB *leveldb.DB
var imageDB *leveldb.DB

func handleCtrlC(c chan os.Signal) {
	sig := <-c

	if parserDB != nil {
		parserDB.Close()
	}

	if imageDB != nil {
		imageDB.Close()
	}

	fmt.Println("\nsignal: ", sig)
	os.Exit(0)
}

// InitCachesDB ...
func InitCachesDB() {
	tempDir := os.TempDir()
	fmt.Println(tempDir)
	parserDBFile := filepath.Join(tempDir, "parser.db")
	imageDBFile := filepath.Join(tempDir, "image.db")
	fmt.Println(parserDBFile)

	var err error
	parserDB, err = leveldb.OpenFile(parserDBFile, nil)
	if err != nil {
		log.Println(err)
	}

	imageDB, err = leveldb.OpenFile(imageDBFile, nil)
	if err != nil {
		log.Println(err)
	}

	//defer parserDB.Close()
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go handleCtrlC(c)
}
