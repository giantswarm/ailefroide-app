package main

import (
	"context"
	"log"
	"time"

	"golang.org/x/oauth2"
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

func NewCalendar(cfg *Config) *GoogleCalendar {
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

	client := conf.Client(oauth2.NoContext)

	g.client, err = calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Calendar client: %v\n", err)
	}

	return &g
}

func (g *GoogleCalendar) GetLocation() (t time.Time, y int, m time.Month, d int) {
	var loc, _ = time.LoadLocation(g.location)

	t = time.Now().In(loc)
	y, m, d = t.Date()
	return
}

func (g *GoogleCalendar) AllDayEvents() []string {
	log.Println("Retrieving all calendar events")
	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, (d + 1), 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
}

func (g *GoogleCalendar) MorningEvents() []string {
	log.Println("Retrieving morning calendar events")
	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, d, 13, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
}

func (g *GoogleCalendar) AfternoonEvents() []string {
	log.Println("Retrieving afternoon calendar events")
	var (
		t, y, m, d = g.GetLocation()
		start      = time.Date(y, m, d, 13, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, d, 18, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
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

func (g *GoogleCalendar) CurrentShiftEvents(events *chan []string) {
	if g.IsMorning() {
		*events <- g.MorningEvents()
		return
	}
	*events <- g.AfternoonEvents()
}

func (g *GoogleCalendar) EventEmailsInTimeSpan(start, end string) (eventEmails []string) {
	events, err := g.client.Events.List(g.calendar).ShowDeleted(false).
		SingleEvents(true).TimeMin(start).TimeMax(end).MaxResults(g.maxentries).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	eventEmails = make([]string, 0)
	if len(events.Items) != 0 {
		for _, item := range events.Items {
			eventEmails = append(eventEmails, item.Creator.Email)
		}
	}
	log.Println("Done retrieving calendar events")
	return
}
