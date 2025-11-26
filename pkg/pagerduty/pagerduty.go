package pagerduty

import (
	"context"
	"log"
	"strings"

	"github.com/PagerDuty/go-pagerduty"
	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
)

type PagerDuty struct {
	client        *pagerduty.Client
	scheduleNames map[string]string
	calendar      *ac.Calendar
}

func New(token string, calendar *ac.Calendar) *PagerDuty {
	o := PagerDuty{
		calendar: calendar,
	}
	var err error
	if o.client = pagerduty.NewClient(token); o.client == nil {
		log.Printf("Unexpected error creating a client for PagerDuty Schedule API: %s", err)
	}

	o.scheduleNames = o.ListSchedules()
	return &o
}

func (o *PagerDuty) ListSchedules() (schedules map[string]string) {
	schedules = make(map[string]string)
	lr := pagerduty.ListSchedulesOptions{}
	sch, _ := o.client.ListSchedulesWithContext(context.Background(), lr)
	for _, item := range sch.Schedules {
		schedules[item.ID] = item.Name
	}
	return
}

// Try and work out who is on call for a given schedule
func (o *PagerDuty) WhoIsOnCall(team *aile.Team) {
	var (
		prefix         = strings.Split(team.Name, "-")[1]
		timeSuffix     = "afternoon"
		scheduleSuffix = "On-Call Schedule"
		schedules      = make([]string, 0)
	)

	if o.calendar.IsMorning() {
		timeSuffix = "morning"
	}

	for id, item := range o.scheduleNames {
		splitSchedule := strings.Join([]string{prefix, timeSuffix}, "_")
		singleSchedule := strings.Join([]string{prefix, scheduleSuffix}, " ")

		if strings.HasPrefix(item, splitSchedule) || item == singleSchedule {
			schedules = append(schedules, id)
		}
	}

	ctx := context.Background()
	opts := pagerduty.ListOnCallOptions{
		ScheduleIDs: schedules,
	}
	ocs, err := o.client.ListOnCallsWithContext(ctx, opts)
	if err != nil {
		for _, scheduleId := range schedules {
			log.Printf("Unexpected error getting on call participants for schedule %s: %s", o.scheduleNames[scheduleId], err)
		}
		return
	}

	for _, p := range ocs.OnCalls {
		opts := pagerduty.GetUserOptions{}
		user, err := o.client.GetUserWithContext(ctx, p.User.ID, opts)
		if err != nil {
			for _, scheduleId := range schedules {
				log.Printf("Error getting oncall user %s for schedule %s: %s", p.User.ID, o.scheduleNames[scheduleId], err)
			}
			return
		}
		for _, m := range team.Members {
			if m.Email == user.Email {
				m.Oncall = o.calendar.IsBusinessHours()
			}
		}
	}
}
