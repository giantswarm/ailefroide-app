package pagerduty

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/PagerDuty/go-pagerduty"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
)

// scheduleCopy matches the per-member escalation-layer duplicates PagerDuty
// creates alongside each schedule ("<name> (1)", "<name> (2)", ...), and the
// older "(Catchup)" variant they replaced. Only the unsuffixed schedule names
// the engineer who is actually on call.
var scheduleCopy = regexp.MustCompile(`\((Catchup|\d+)\)$`)

type PagerDuty struct {
	client        *pagerduty.Client
	scheduleNames map[string]string
	calendar      *ac.Calendar
}

func New(token string, calendar *ac.Calendar) (*PagerDuty, error) {
	o := PagerDuty{
		calendar: calendar,
	}
	if o.client = pagerduty.NewClient(token); o.client == nil {
		return nil, fmt.Errorf("failed to create a client for the PagerDuty Schedule API")
	}

	var err error
	if o.scheduleNames, err = o.ListSchedules(); err != nil {
		return nil, err
	}
	return &o, nil
}

// ListSchedules maps every schedule ID to its name.
//
// The escalation ladder gives each schedule one copy per team member, so the
// account holds far more than the 25 schedules a single unpaginated call
// returns - page through the lot.
func (o *PagerDuty) ListSchedules() (map[string]string, error) {
	schedules := make(map[string]string)
	lr := pagerduty.ListSchedulesOptions{Limit: 100}
	for {
		sch, err := o.client.ListSchedulesWithContext(context.Background(), lr)
		if err != nil {
			return nil, fmt.Errorf("listing PagerDuty schedules at offset %d: %w", lr.Offset, err)
		}
		for _, item := range sch.Schedules {
			schedules[item.ID] = item.Name
		}
		if !sch.More {
			return schedules, nil
		}
		lr.Offset += lr.Limit
	}
}

// matchSchedules picks out the IDs of the schedules covering team for the
// current shift, ignoring the escalation-layer copies.
func matchSchedules(scheduleNames map[string]string, team string, morning bool) []string {
	var (
		prefix         = strings.Split(team, "-")[1]
		timeSuffix     = "afternoon"
		scheduleSuffix = "On-Call Schedule"
		schedules      = make([]string, 0)
	)

	if morning {
		timeSuffix = "morning"
	}

	splitSchedule := strings.Join([]string{prefix, timeSuffix}, "_")
	singleSchedule := strings.Join([]string{prefix, scheduleSuffix}, " ")

	for id, item := range scheduleNames {
		if scheduleCopy.MatchString(item) {
			continue
		}

		if strings.HasPrefix(item, splitSchedule) || item == singleSchedule {
			log.Println("Checking schedule ", item)
			schedules = append(schedules, id)
		}
	}
	return schedules
}

// Try and work out who is on call for a given schedule
func (o *PagerDuty) WhoIsOnCall(team *aile.Team) {
	schedules := matchSchedules(o.scheduleNames, team.Name, o.calendar.IsMorning())

	// An empty ScheduleIDs filter asks PagerDuty for every on-call in the
	// account, which would mark unrelated engineers as on call for this team.
	if len(schedules) == 0 {
		log.Printf("No PagerDuty schedule matches team %s, skipping on-call lookup", team.Name)
		return
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
