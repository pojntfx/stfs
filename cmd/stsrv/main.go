package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/pojntfx/stfs/internal/fs"
	"github.com/pojntfx/stfs/internal/handlers"
	"github.com/spf13/afero"
)

func main() {
	laddr := flag.String("laddr", "localhost:1337", "Listen address")
	dir := flag.String("dir", "/", "Directory to use as the root directory")

	flag.Parse()

	stfs := afero.NewHttpFs(fs.NewSTFS())

	log.Println("Listening on", *laddr)

	log.Fatal(http.ListenAndServe(*laddr, handlers.PanicHandler(http.FileServer(stfs.Dir(*dir)))))
}
