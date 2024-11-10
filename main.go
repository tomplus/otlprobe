package main

import (
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var screen tcell.Screen

var bucket Bucket = newBucketFixedSize(100)

func main() {

	grpcPortPtr := flag.Int("grpc-port", 4317, "port for gRPC server (default 4317)")
	grpcDisablePtr := flag.Bool("disable-grpc", false, "disable gRPC server")
	httpPortPtr := flag.Int("http-port", 4318, "port for HTTP server (default 4318)")
	httpDisablePtr := flag.Bool("disable-http", false, "disable HTTP server")
	filterPtr := flag.String("filter", "", "filter for incomming data")
	noninteractivePtr := flag.Bool("non-interactive", false, "print out data to stdout (without TUI)")
	flag.Parse()

	grpcPort, httpPort := 0, 0
	if !*grpcDisablePtr {
		grpcPort = *grpcPortPtr
	}
	if !*httpDisablePtr {
		httpPort = *httpPortPtr
	}
	if grpcPort == 0 && httpPort == 0 {
		log.Fatalln("Disabled gRPC and HTTP")
	}
	if grpcPort < 0 || httpPort < 0 {
		log.Fatalln("Invalid port number")
	}

	chSignal := make(chan *Signal)
	server := newServer(grpcPort, httpPort, chSignal)
	go server.start()

	if *noninteractivePtr {
		i := 0
		for c := range chSignal {
			i++
			if strings.Contains(c.description, *filterPtr) || strings.Contains(c.summary, *filterPtr) {
				fmt.Printf("%v: %v %v\n", i, c.time.AsTime().String()[0:23], c.summary)
			}
		}
		return
	}

	s, err := tcell.NewScreen()

	screen = s
	if err != nil {
		log.Fatalf("%+v", err)
	}
	if err := s.Init(); err != nil {
		log.Fatalf("%+v", err)
	}
	s.EnableMouse()
	s.EnablePaste()
	s.Clear()

	browser := newBrowser(screen, bucket, *filterPtr, chSignal)
	browser.refresh()
	// go genRandomData(browser.ch)

	quit := func() {
		maybePanic := recover()
		s.Fini()
		if maybePanic != nil {
			panic(maybePanic)
		}
	}
	defer quit()

	for {
		s.Show()
		ev := s.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			browser.resize()
			s.Sync()
		case *tcell.EventKey:
			res := browser.eventKey(ev)
			if !res {
				if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
					return
				}
			}
		}
	}
}
