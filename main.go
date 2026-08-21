package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"task142-atcconflict/internal/httpapi"
	"task142-atcconflict/internal/selfcheck"
	"task142-atcconflict/internal/service"
	"task142-atcconflict/internal/store"
	"task142-atcconflict/internal/webfs"
)

func main() {
	smoke := flag.Bool("smoke-test", false, "run the deterministic engine smoke test")
	addr := flag.String("addr", ":8080", "HTTP listen address")
	flag.Parse()
	if *smoke {
		if err := selfcheck.Run(); err != nil {
			log.Fatal(err)
		}
		fmt.Println("smoke-test: ok")
		return
	}
	st, err := store.Open("atc.db")
	if err != nil {
		log.Fatal(err)
	}
	defer st.Close()
	mux := httpapi.New(service.New(st, nil), webfs.FS())
	log.Printf("ATC conflict engine listening on %s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}
