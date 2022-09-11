package main

import (
	"regexp"
	"strings"
	"time"
)

func inTimeSpan(s, e string, t time.Time) bool {
	var (
		start, _ = time.Parse("15:04", s)
		end, _   = time.Parse("15:04", e)
	)
	if start.Before(end) {
		return !t.Before(start) && !t.After(end)
	}
	if start.Equal(end) {
		return t.Equal(start)
	}
	return !start.After(t) || !end.Before(t)
}

func containsString(what string, where []string) bool {
	for _, i := range where {
		if i == what {
			return true
		}
	}
	return false
}

func looksLikeATopic(what string) bool {
	re, _ := regexp.Compile(`^(\w+[ -]?){1,2}$`)
	return re.Match([]byte(what))
}

func appendTopic(supportTopics map[string][]string, topic, team string) map[string][]string {
	if looksLikeATopic(topic) {
		topic = strings.ToLower("support-" + strings.ReplaceAll(topic, " ", "-"))
		for k := range supportTopics {
			if k == topic {
				if !containsString(team, supportTopics[k]) {
					supportTopics[k] = append(supportTopics[k], team)
				}
				return supportTopics
			}
		}
		supportTopics[topic] = make([]string, 0)
		supportTopics[topic] = append(supportTopics[topic], team)
	}
	return supportTopics
}
