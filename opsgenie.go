package main

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
)

type Opsgenie struct {
	client        *schedule.Client
	scheduleNames []string
}

func NewOpsGenie(token string) *Opsgenie {
	o := Opsgenie{}
	var err error
	if o.client, err = schedule.NewClient(&client.Config{
		ApiKey: token,
	}); err != nil {
		log.Printf("Unexpected creating a client for Opsgenie Schedule API: %s", err)
	}

	o.scheduleNames = o.ListSchedules()
	return &o
}

func (o *Opsgenie) inTimeSpan(start, end, check time.Time) bool {
	if start.Before(end) {
		return !check.Before(start) && !check.After(end)
	}
	if start.Equal(end) {
		return check.Equal(start)
	}
	return !start.After(check) || !end.Before(check)
}

func (o *Opsgenie) ListSchedules() (schedules []string) {
	schedules = make([]string, 0)
	var expand bool = false
	lr := schedule.ListRequest{
		Expand: &expand,
	}
	sch, _ := o.client.List(context.Background(), &lr)
	for _, item := range sch.Schedule {
		schedules = append(schedules, item.Name)
	}
	return
}

// Try and work out who is on call for a given schedule
func (o *Opsgenie) WhoIsOnCall(team *Team) {
	var (
		prefix         string = strings.Split(team.Name, "-")[1]
		timeSuffix     string = "am"
		scheduleSuffix string = "schedule"

		start, _           = time.Parse("15:04", "13:00")
		end, _             = time.Parse("15:04", "23:59")
		schedules []string = make([]string, 0)
	)

	if o.inTimeSpan(start, end, time.Now()) {
		timeSuffix = "pm"
	}

	for _, item := range o.scheduleNames {
		if strings.HasPrefix(item, prefix) && (strings.HasSuffix(item, timeSuffix) || strings.HasSuffix(item, scheduleSuffix)) {
			schedules = append(schedules, item)
		}
	}

	for _, scheduleName := range schedules {
		flat := false
		date := time.Now()
		scheduleResult, err := o.client.GetOnCalls(context.TODO(), &schedule.GetOnCallsRequest{
			Flat:                   &flat,
			Date:                   &date,
			ScheduleIdentifierType: schedule.Name,
			ScheduleIdentifier:     scheduleName,
		})
		if err != nil {
			continue
		}

		p := scheduleResult.OnCallParticipants
		if len(p) > 0 {
			for _, m := range team.Members {
				if m.Email == p[0].Name {
					m.Oncall = true
				}
			}
		}
	}
}
