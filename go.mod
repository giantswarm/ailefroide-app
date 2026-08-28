module github.com/giantswarm/ailefroide

go 1.25.0

require (
	github.com/PagerDuty/go-pagerduty v1.8.0
	github.com/bradleyfalzon/ghinstallation/v2 v2.19.0
	github.com/creasty/defaults v1.7.0
	github.com/giantswarm/personio-go v0.6.0
	github.com/google/go-github/v88 v88.0.0
	github.com/slack-go/slack v0.23.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/golang-jwt/jwt/v4 v4.5.2 // indirect
	github.com/google/go-querystring v1.2.0 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
)

replace golang.org/x/sys v0.15.0 => golang.org/x/sys v0.47.0
