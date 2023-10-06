package calendar

import (
	"context"
	"log"
	"time"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type GoogleCalendar struct {
	client     *calendar.Service
	calendar   string
	location   string
	maxentries int64
}

func NewCalendar(cfg *aile.Config) *GoogleCalendar {
	g := GoogleCalendar{
		calendar:   cfg.AfkCalendar,
		location:   cfg.Location,
		maxentries: int64(cfg.PagingEntries),
	}
	ctx := context.Background()
	conf, err := google.JWTConfigFromJSON(cfg.CalendarCredentials, calendar.CalendarReadonlyScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}

	client := conf.Client(context.Background())

	g.client, err = calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v\n", err)
	}

	log.Printf("Using location '%s' for calendar entries", g.location)
	return &g
}

func (g *GoogleCalendar) GetLocation() (t time.Time, y int, m time.Month, d int) {
	var loc, _ = time.LoadLocation(g.location)

	t = time.Now().In(loc)
	y, m, d = t.Date()
	return
}

func (g *GoogleCalendar) IsMorning() bool {
	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location())
		end        = time.Date(y, m, d, 13, 0, 0, 0, t.Location())
		_start     = start
		_end       = end
		_check     = t
	)
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
