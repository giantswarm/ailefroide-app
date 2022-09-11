package main

import (
	"context"
	"log"
	"os"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

const (
	AFK_CALENDAR = "giantswarm.io_u9j5eaid81sl9b8cd73novr7do@group.calendar.google.com"
	MAX_ENTRIES  = 200
	LOCATION     = "Europe/Berlin"
)

type GoogleCalendar struct {
	client *calendar.Service
}

func NewCalendar() *GoogleCalendar {
	g := GoogleCalendar{}
	ctx := context.Background()
	b, err := os.ReadFile("token.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	conf, err := google.JWTConfigFromJSON(b, calendar.CalendarReadonlyScope)
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

func (g *GoogleCalendar) location() (t time.Time, y int, m time.Month, d int) {
	var loc, _ = time.LoadLocation(LOCATION)

	t = time.Now().In(loc)
	y, m, d = t.Date()
	return
}

func (g *GoogleCalendar) AllDayEvents() []string {
	var (
		t, y, m, d = g.location()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, (d + 1), 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
}

func (g *GoogleCalendar) MorningEvents() []string {
	var (
		t, y, m, d = g.location()
		start      = time.Date(y, m, d, 0, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, d, 13, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
}

func (g *GoogleCalendar) AfternoonEvents() []string {
	var (
		t, y, m, d = g.location()
		start      = time.Date(y, m, d, 13, 0, 0, 0, t.Location()).Format(time.RFC3339)
		end        = time.Date(y, m, d, 18, 0, 0, 0, t.Location()).Format(time.RFC3339)
	)
	return g.EventEmailsInTimeSpan(start, end)
}

func (g *GoogleCalendar) CurrentShiftEvents() []string {
	if inTimeSpan("00:00", "13:00", time.Now()) {
		return g.MorningEvents()
	}
	return g.AfternoonEvents()
}

func (g *GoogleCalendar) EventEmailsInTimeSpan(start, end string) (eventEmails []string) {
	events, err := g.client.Events.List(AFK_CALENDAR).ShowDeleted(false).
		SingleEvents(true).TimeMin(start).TimeMax(end).MaxResults(MAX_ENTRIES).OrderBy("startTime").Do()
	if err != nil {
		log.Fatalf("Unable to retrieve next ten of the user's events: %v", err)
	}

	eventEmails = make([]string, 0)
	if len(events.Items) != 0 {
		for _, item := range events.Items {
			eventEmails = append(eventEmails, item.Creator.Email)
		}
	}
	return
}
