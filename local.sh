export GITHUB_TOKEN=$(bwv "development/github.com?field=full-access-token-never-expire" | jq -r .value);
export SLACK_TOKEN=$(bwv '*/ailefroide-slack?property=password' | jq -r .value)
export OPSGENIE_TOKEN=$(bwv '*/opsgenie?field=apikey' | jq -r .value)
export PERSONIO_CLIENT_ID=$(bwv '*/personio?field=clientid'| jq -r .value)
export PERSONIO_CLIENT_SECRET=$(bwv '*/personio?field=clientsecret'| jq -r .value)

go build .
time ./ailefroide -config config.yaml -debugteam "team-honeybadger" -debug true
