# ailefroide-app

A Helm chart for Ailefroide, the Slack support handle manager

**Homepage:** <https://github.com/giantswarm/ailefroide-app>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Giant Swarm applications team |  | <https://github.com/giantswarm/ailefroide-app> |

## Source Code

* <https://github.com/giantswarm/ailefroide-app>
* <https://raw.githubusercontent.com/giantswarm/ailefroide-app/v[[ .Version ]]/README.md>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| debug | bool | `true` |  |
| envSecret.name | string | `"ailefroide-app-environment"` |  |
| envSecret.values.SLACK_TOKEN | string | `""` |  |
| envSecret.values.OPSGENIE_TOKEN | string | `""` |  |
| envSecret.values.PERSONIO_CLIENT_ID | string | `""` |  |
| envSecret.values.PERSONIO_CLIENT_SECRET | string | `""` |  |
| configMap.name | string | `"ailefroide-config"` |  |
| configMap.values."config.yaml" | string | `""` |  |
| configMap.values."token.json" | string | `""` |  |
| configMap.values."github.pem" | string | `""` |  |
| image.repository | string | `"gsoci.azurecr.io/giantswarm/ailefroide"` |  |
| image.pullPolicy | string | `"Always"` |  |
| image.tag | string | `""` |  |
| imagePullSecrets | list | `[]` |  |
| nameOverride | string | `""` |  |
| fullnameOverride | string | `""` |  |
| serviceAccount.create | bool | `true` |  |
| serviceAccount.name | string | `nil` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.runAsUser | int | `1000` |  |
| podSecurityContext.runAsGroup | int | `1000` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.runAsUser | int | `1000` |  |
| securityContext.runAsGroup | int | `3000` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| resources.limits.cpu | string | `"250m"` |  |
| resources.limits.memory | string | `"256Mi"` |  |
| resources.requests.cpu | string | `"250m"` |  |
| resources.requests.memory | string | `"256Mi"` |  |
| nodeSelector | object | `{}` |  |
| tolerations | list | `[]` |  |
| affinity | object | `{}` |  |
| rotation.schedule | string | `"0 * * * 1-5"` |  |
