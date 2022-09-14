package ailefroide

import (
	"regexp"
	"strings"
)

func ContainsString(what string, where []string) bool {
	for _, i := range where {
		if i == what {
			return true
		}
	}
	return false
}

func AppendTopic(supportTopics map[string][]string, topic, team string) map[string][]string {
	if looksLikeATopic(topic) {
		topic = strings.ToLower("support-" + strings.ReplaceAll(topic, " ", "-"))
		for k := range supportTopics {
			if k == topic {
				if !ContainsString(team, supportTopics[k]) {
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

func looksLikeATopic(what string) bool {
	re, _ := regexp.Compile(`^(\w+[ -]?){1,2}$`)
	return re.Match([]byte(what))
}
