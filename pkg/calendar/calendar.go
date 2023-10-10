package calendar

import (
	"log"
	"strconv"
	"strings"
	"time"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
)

type Calendar struct {
	calendar   string
	location   string
	maxentries int64
	cfg        *aile.Config
}

func NewCalendar(cfg *aile.Config) *Calendar {
	g := Calendar{
		calendar:   cfg.AfkCalendar,
		location:   cfg.Location,
		maxentries: int64(cfg.PagingEntries),
		cfg:        cfg,
	}

	log.Printf("Using location '%s' for calendar entries", g.location)
	return &g
}

func (g *Calendar) GetLocation() (t time.Time, y int, m time.Month, d int) {
	var loc, _ = time.LoadLocation(g.location)

	t = time.Now().In(loc)
	y, m, d = t.Date()
	return
}

func (g *Calendar) IsMorning() bool {

	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		end        time.Time
		_start         = start
		_end           = end
		_check         = t
		ah, am     int = g.getTimeAsHoursMinutes(g.cfg.MiddayShiftChange)
	)

	if ah == -1 {
		ah = 13
	}
	end = time.Date(y, m, d, ah, am, 0, 0, t.Location())

	if end.Before(start) {
		_end = end.Add(24 * time.Hour)
		if t.Before(start) {
			_check = t.Add(24 * time.Hour)
		}
	}

	_start = _start.Add(-1 * time.Nanosecond)
	_end = _end.Add(1 * time.Nanosecond)

	return _check.After(_start) && _check.Before(_end)
}

func (g *Calendar) Morning() (start, end time.Time) {
	var (
		t, y, m, d = g.GetLocation()
		h, mi      = g.getTimeAsHoursMinutes(g.cfg.MiddayShiftChange)
	)
	if h == -1 {
		h = 0
	}
	start = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	end = time.Date(y, m, d, h, mi, 0, 0, t.Location())
	return
}

func (g *Calendar) Afternoon() (start, end time.Time) {
	var (
		t, y, m, d = g.GetLocation()
		h, mi      = g.getTimeAsHoursMinutes(g.cfg.MiddayShiftChange)
	)
	if h == -1 {
		h = 13
	}
	start = time.Date(y, m, d, h, mi, 0, 0, t.Location())
	end = time.Date(y, m, d, 23, 59, 0, 0, t.Location())
	return
}

func (g *Calendar) CurrentShift() (start, end time.Time) {
	if g.IsMorning() {
		return g.Morning()
	}
	return g.Afternoon()
}

func (g *Calendar) getTimeAsHoursMinutes(t string) (int, int) {
	var (
		h, m int
		x    []string = strings.Split(t, ":")
		e    error
	)
	if h, e = strconv.Atoi(x[0]); e != nil {
		h = -1
	}
	if m, e = strconv.Atoi(x[1]); e != nil {
		m = 0
	}
	return h, m
}
