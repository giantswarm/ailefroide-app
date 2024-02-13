package calendar

import (
	"log"
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

// GetLocation returns the current time in the configured location (default: Europe/Berlin)
func (g *Calendar) GetLocation() (t time.Time, y int, m time.Month, d int) {
	var loc, _ = time.LoadLocation(g.location)

	t = time.Now().In(loc)
	y, m, d = t.Date()
	return
}

// IsMorning returns true if the current time is between 0am and the configured
// midday shift change time (default: 1pm)
func (g *Calendar) IsMorning() bool {

	var (
		current, _, _, _ = g.GetLocation()
		start, end       = g.Morning()
	)

	return current.Equal(start) || (current.After(start) && current.Before(end))
}

// Morning returns the start and end time of the morning shift
//
// To prevent Ailefroide from failing to find the correct calendar entries
// the morning shift is configured to start from midnight to the configured
// midday shift change time (default: 1pm)
func (g *Calendar) Morning() (start, end time.Time) {
	var (
		t, y, m, d = g.GetLocation()
		h, mi      = g.getMiddayShiftChange()
	)

	start = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
	end = time.Date(y, m, d, h, mi, 0, 0, t.Location())
	return
}

// Afternoon returns the start and end time of the afternoon shift running from
// the configured midday shift change time (default: 1pm) to midnight (0am)
func (g *Calendar) Afternoon() (start, end time.Time) {
	var (
		t, y, m, d = g.GetLocation()
		h, mi      = g.getMiddayShiftChange()
	)
	start = time.Date(y, m, d, h, mi, 0, 0, t.Location())
	end = time.Date(y, m, d, 23, 59, 0, 0, t.Location())
	return
}

// IsBusinessHours returns true if the current time is between 9am and 6pm
func (g *Calendar) IsBusinessHours() bool {
	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 9, 0, 0, 0, t.Location())
		end        = time.Date(y, m, d, 18, 0, 0, 0, t.Location())
	)
	return t.After(start) && t.Before(end)
}

// CurrentShift returns the start and end time of the current shift
func (g *Calendar) CurrentShift() (start, end time.Time) {
	if g.IsMorning() {
		return g.Morning()
	}
	return g.Afternoon()
}

func (g *Calendar) getMiddayShiftChange() (int, int) {
	var (
		h, m int = 13, 0 // default to 13:00
	)

	if current, err := time.Parse("15:04", g.cfg.MiddayShiftChange); err == nil {
		h, m = current.Hour(), current.Minute()
	}

	return h, m
}
