package main

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
)

type Browser struct {
	screen      tcell.Screen
	bucket      Bucket
	cursor      int
	width       int
	height      int
	follow      bool
	inputFilter bool
	filter      string

	ch chan *Signal

	hb    *HeartbeatWidget
	popUp *PopUp

	rowStyle             tcell.Style
	rowSelectedStyle     tcell.Style
	statusStyle          tcell.Style
	statusHighlightStyle tcell.Style
}

func newBrowser(screen tcell.Screen, bucket Bucket, filter string, ch chan *Signal) *Browser {
	w, h := screen.Size()
	b := Browser{
		screen:               screen,
		bucket:               bucket,
		cursor:               -1,
		width:                w,
		height:               h,
		follow:               true,
		filter:               filter,
		ch:                   ch,
		hb:                   newHeartbeatWidget(screen),
		popUp:                newPopUp(screen),
		rowStyle:             tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack),
		rowSelectedStyle:     tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorWhite).Bold(true),
		statusStyle:          tcell.StyleDefault.Foreground(tcell.ColorBlack).Background(tcell.ColorLightCyan),
		statusHighlightStyle: tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorDarkCyan),
	}

	go func() {
		for c := range b.ch {
			b.hb.tick()
			if b.follow {
				if strings.Contains(c.description, b.filter) || strings.Contains(c.summary, b.filter) {
					bucket.append(c)
					b.refresh()
				}
			}

		}
	}()

	return &b
}

func (browser *Browser) resize() {
	w, h := screen.Size()
	if w != browser.width || h != browser.height {
		browser.width = w
		browser.height = h
		browser.refresh()
	}
	browser.popUp.resize()
}

func (browser *Browser) drawRow(y int, style tcell.Style, highlight tcell.Style, text string) {

	var hl []int = make([]int, browser.width)
	if browser.filter != "" {
		p := 0
		for {
			f := strings.Index(text[p:], browser.filter)
			if f == -1 {
				break
			}
			p += f
			for i := 0; i < len(browser.filter); i++ {
				if p > browser.width {
					// match in invisible part, mark last character
					// and escape from loops
					hl[browser.width-1] = 1
					p = len(text)
					break
				}
				hl[p] = 1
				p++
			}
		}
	}

	for i, col := 0, 0; col < browser.width; col++ {
		r := ' '
		if i < len(text) {
			var s int
			r, s = utf8.DecodeRuneInString(text[i:])
			i += s
		}
		s := style
		if hl[col] == 1 {
			s = highlight
		}
		browser.screen.SetContent(col, y, r, nil, s)
	}
}

func (browser *Browser) drawText(x int, y int, style tcell.Style, text string) {
	col := x
	for i := 0; i < len(text); {
		runeValue, width := utf8.DecodeRuneInString(text[i:])
		browser.screen.SetContent(col, y, runeValue, nil, style)
		i += width
		col++
		if col >= browser.width {
			return
		}
	}
}

func (browser *Browser) refreshStatusBar() {

	filter := "filter"
	if browser.filter != "" {
		filter += " [" + browser.filter + "]"
	}

	menu := []string{"↑↓", "select", "Enter", "details", "Esc", "exit", "Shift+F", "follow", "/", filter}
	if browser.follow {
		menu = []string{"↑↓", "stop & select", "Esc", "stop following", "/", filter}
	}
	if browser.inputFilter {
		menu = []string{"Find", browser.filter}
	}

	col := 4
	for i := 0; i < len(menu); i += 2 {
		browser.drawText(col, browser.height-1, browser.statusHighlightStyle, " "+menu[i]+" ")
		col += utf8.RuneCountInString(menu[i]) + 2
		browser.drawText(col, browser.height-1, browser.statusStyle, " "+menu[i+1]+"  ")
		col += len(menu[i+1]) + 3
	}

	if browser.inputFilter {
		browser.screen.ShowCursor(col-2, browser.height-1)
	} else {
		browser.screen.HideCursor()
	}

	if col < browser.width {
		browser.drawText(col, browser.height-1, browser.statusStyle, strings.Repeat(" ", browser.width-col))
	}

}

func (browser *Browser) eventKey(ev *tcell.EventKey) bool {

	if browser.popUp.visible {
		h := browser.popUp.eventKey(ev)
		if h {
			browser.refresh()
		}
		return h
	}

	if ev.Key() == tcell.KeyDown {
		if browser.cursor > 0 {
			browser.cursor--
		} else {
			browser.cursor = 0
		}
		browser.follow = false
		browser.refresh()
		return true
	} else if ev.Key() == tcell.KeyUp {
		rng := browser.bucket.len() - 1
		if browser.height-2 < rng {
			rng = browser.height - 2
		}
		if browser.cursor < rng {
			browser.cursor++
		} else {
			browser.cursor = rng
		}
		browser.follow = false
		browser.refresh()
		return true
	} else if ev.Rune() == 'F' && !browser.follow {
		browser.follow = true
		browser.cursor = -1
		browser.refresh()
		return true
	} else if ev.Key() == tcell.KeyEscape {
		if browser.inputFilter {
			browser.inputFilter = false
			browser.refresh()
			return true
		} else if browser.follow {
			browser.follow = false
			browser.cursor = -1
			browser.refresh()
			return true
		}
	} else if ev.Rune() == '/' && !browser.inputFilter {
		browser.inputFilter = true
		browser.refresh()
		return true
	} else if ev.Key() == tcell.KeyEnter {
		if browser.inputFilter {
			browser.inputFilter = false
			browser.refresh()
			return true
		} else if browser.cursor != -1 {
			ok, data := browser.bucket.get(browser.cursor)
			if ok {
				browser.popUp.show(data.properties)
			}
		}
	} else if ev.Key() == tcell.KeyBackspace2 && browser.inputFilter {
		if len(browser.filter) > 0 {
			browser.filter = browser.filter[0 : len(browser.filter)-1]
			browser.refresh()
		}

	} else if !unicode.IsControl(ev.Rune()) && browser.inputFilter {
		browser.filter += string(ev.Rune())
		browser.refresh()
		return true
	}
	return false
}

func (browser *Browser) refresh() {

	browser.popUp.refresh()

	if !browser.popUp.visible {
		for i, j := 0, browser.height-2; j >= 0; j-- {
			style := browser.rowStyle
			ok, val := bucket.get(i)
			if ok {
				if i == browser.cursor {
					style = browser.rowSelectedStyle
				}
				browser.drawRow(j, style, browser.rowSelectedStyle, val.summary)
			} else {
				browser.drawRow(j, style, browser.rowSelectedStyle, "")
			}
			i++
		}
	}

	browser.refreshStatusBar()
	screen.Sync()

}
