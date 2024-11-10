package main

import (
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type HeartbeatWidget struct {
	screen  tcell.Screen
	style   tcell.Style
	counter atomic.Uint64
}

func newHeartbeatWidget(screen tcell.Screen) *HeartbeatWidget {
	w := HeartbeatWidget{
		screen: screen,
		style:  tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightCyan),
	}
	go w.beat()
	return &w
}

func (hbw *HeartbeatWidget) beat() {

	ind := []string{"🭶", "🭷", "🭸", "🭹", "🭺", "🭻", "🭺", "🭹", "🭸", "🭷"}

	for counter, step := uint64(0), 0; ; {

		nc := hbw.counter.Load()
		if nc != counter {
			step++
			counter = nc
			active := ind[step%len(ind)]
			_, h := hbw.screen.Size()
			runeValue, _ := utf8.DecodeRuneInString(active[0:])
			hbw.screen.SetContent(0, h-1, ' ', nil, hbw.style)
			hbw.screen.SetContent(1, h-1, runeValue, nil, hbw.style)
			hbw.screen.SetContent(2, h-1, runeValue, nil, hbw.style)
			hbw.screen.SetContent(3, h-1, ' ', nil, hbw.style)
			hbw.screen.Sync()
		}
		time.Sleep(250 * time.Millisecond)

	}

}

func (hbw *HeartbeatWidget) tick() {
	hbw.counter.Add(1)
}
