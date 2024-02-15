export GITHUB_TOKEN=$(bwv "development/github.com" -f full-access-token | jq -r .value);
export SLACK_TOKEN=$(bwv '*/ailefroide-slack?properties=password' | jq -r .value)
export OPSGENIE_TOKEN=$(bwv '*/opsgenie?fields=apikey' | jq -r .value)
export PERSONIO_CLIENT_ID=$(bwv '*/personio?fields=clientid'| jq -r .value)
export PERSONIO_CLIENT_SECRET=$(bwv '*/personio?fields=clientsecret'| jq -r .value)

go build .
time ./ailefroide -config examples/config.yaml -debug true
