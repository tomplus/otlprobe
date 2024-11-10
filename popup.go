package main

import (
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type PopUp struct {
	screen         tcell.Screen
	data           []Properties
	x0, y0, x1, y1 int
	visible        bool
	frameStyle     tcell.Style
	textStyle      tcell.Style
}

func newPopUp(screen tcell.Screen) *PopUp {
	b := PopUp{
		screen:     screen,
		frameStyle: tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkGray),
		textStyle:  tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorDarkGray),
	}
	b.center()
	return &b
}

func (popUp *PopUp) show(data []Properties) {
	popUp.data = data
	popUp.visible = true
	popUp.refresh()
}

func (popUp *PopUp) center() {
	w, h := screen.Size()
	wm := int(math.Floor(float64(w) * 0.1 * 0.5))
	hm := int(math.Floor(float64(h) * 0.1 * 0.5))
	if wm < 2 {
		wm = 2
	}
	if hm < 2 {
		hm = 2
	}
	if w-wm < 0 {
		wm = 0
	}
	if h-hm < 0 {
		hm = 0
	}
	popUp.x0 = wm
	popUp.y0 = hm
	popUp.x1 = w - wm
	popUp.y1 = h - hm
}

func (popUp *PopUp) resize() {
	w, h := screen.Size()
	if w != popUp.x1-5 || h != popUp.y1-5 {
		popUp.x1 = w - 5
		popUp.y1 = h - 5
		popUp.refresh()
	}
}

func (popUp *PopUp) refresh() {

	if !popUp.visible {
		return
	}

	for col := popUp.x0; col <= popUp.x1; col++ {
		popUp.screen.SetContent(col, popUp.y0, tcell.RuneHLine, nil, popUp.frameStyle)
		popUp.screen.SetContent(col, popUp.y1, tcell.RuneHLine, nil, popUp.frameStyle)
	}
	for row := popUp.y0 + 1; row < popUp.y1; row++ {
		popUp.screen.SetContent(popUp.x0, row, tcell.RuneVLine, nil, popUp.frameStyle)
		popUp.screen.SetContent(popUp.x1, row, tcell.RuneVLine, nil, popUp.frameStyle)
	}
	if popUp.y0 != popUp.y1 && popUp.x0 != popUp.x1 {
		popUp.screen.SetContent(popUp.x0, popUp.y0, tcell.RuneULCorner, nil, popUp.frameStyle)
		popUp.screen.SetContent(popUp.x1, popUp.y0, tcell.RuneURCorner, nil, popUp.frameStyle)
		popUp.screen.SetContent(popUp.x0, popUp.y1, tcell.RuneLLCorner, nil, popUp.frameStyle)
		popUp.screen.SetContent(popUp.x1, popUp.y1, tcell.RuneLRCorner, nil, popUp.frameStyle)
	}

	lines := make([]string, 1)
	for i := 0; i < len(popUp.data); i++ {
		prop := popUp.data[i]
		lines = append(lines, fmt.Sprintf(" + %s", prop.Name()))
		row := prop.get()
		for j := 0; j < len(row); j++ {
			lines = append(lines, fmt.Sprintf(" | %s: %s", row[j][0], row[j][1]))
		}
		lines = append(lines, "")
	}

	lf := false
	for tl, tc, row := 0, 0, popUp.y0+1; row < popUp.y1; row++ {
		for col := popUp.x0 + 1; col < popUp.x1; col++ {

			r := ' '
			if len(lines) > tl && len(lines[tl]) > tc {
				var s int
				r, s = utf8.DecodeRuneInString(lines[tl][tc:])
				tc += s
			} else {
				lf = true
			}

			popUp.screen.SetContent(col, row, r, nil, popUp.textStyle)
		}
		if lf {
			lf = false
			tl++
			tc = 0
		}
	}

}

func (popUp *PopUp) eventKey(ev *tcell.EventKey) bool {

	if ev.Key() == tcell.KeyEscape {
		popUp.visible = false
		return true
	}

	return false
}
