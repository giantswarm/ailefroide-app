package opsgenie

import (
	"context"
	"log"
	"strings"

	aile "github.com/giantswarm/ailefroide/pkg/ailefroide"
	ac "github.com/giantswarm/ailefroide/pkg/calendar"
	"github.com/opsgenie/opsgenie-go-sdk-v2/client"
	"github.com/opsgenie/opsgenie-go-sdk-v2/schedule"
)

type Opsgenie struct {
	client        *schedule.Client
	scheduleNames []string
	calendar      *ac.GoogleCalendar
}

func NewOpsGenie(token string, calendar *ac.GoogleCalendar) *Opsgenie {
	o := Opsgenie{
		calendar: calendar,
	}
	var err error
	if o.client, err = schedule.NewClient(&client.Config{
		ApiKey: token,
	}); err != nil {
		log.Printf("Unexpected creating a client for Opsgenie Schedule API: %s", err)
	}

	o.scheduleNames = o.ListSchedules()
	return &o
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
func (o *Opsgenie) WhoIsOnCall(team *aile.Team) {
	var (
		prefix         string   = strings.Split(team.Name, "-")[1]
		timeSuffix     string   = "pm"
		scheduleSuffix string   = "schedule"
		schedules      []string = make([]string, 0)
	)

	if o.calendar.IsMorning() {
		timeSuffix = "am"
	}

	for _, item := range o.scheduleNames {
		if strings.HasPrefix(item, prefix) && (strings.HasSuffix(item, timeSuffix) || strings.HasSuffix(item, scheduleSuffix)) {
			schedules = append(schedules, item)
		}
	}

	for _, scheduleName := range schedules {
		var (
			flat           bool = false
			scheduleResult *schedule.GetOnCallsResult
			err            error
		)

		if scheduleResult, err = o.client.GetOnCalls(context.TODO(), &schedule.GetOnCallsRequest{
			Flat:                   &flat,
			ScheduleIdentifierType: schedule.Name,
			ScheduleIdentifier:     scheduleName,
		}); err != nil {
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
