# ECS Task Observer

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [unmaintained]  |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aextension%2Fecstaskobserver%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aextension%2Fecstaskobserver) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aextension%2Fecstaskobserver%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aextension%2Fecstaskobserver) |
| Code coverage | [![codecov](https://codecov.io/github/open-telemetry/opentelemetry-collector-contrib/graph/main/badge.svg?component=extension_ecs_task_observer)](https://app.codecov.io/gh/open-telemetry/opentelemetry-collector-contrib/tree/main/?components%5B0%5D=extension_ecs_task_observer&displayType=list) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    |  |
| Emeritus      | [@rmfitzpatrick](https://www.github.com/rmfitzpatrick) |

[unmaintained]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#unmaintained
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

The `ecs_task_observer` is a [Receiver Creator](../../../receiver/receivercreator/README.md)-compatible "watch observer" that will detect and report
container endpoints for the running ECS task of which your Collector instance is a member. It is designed for and only supports "sidecar" deployments
to detect co-located containers. For cluster wide use cases you should use the [ECS Observer](../ecsobserver/README.md) with a corresponding Prometheus receiver.

The Observer works by querying the available [task metadata endpoint](https://docs.aws.amazon.com/AmazonECS/latest/developerguide/task-metadata-endpoint.html)
and making all detected running containers available as endpoints for Receiver Creator usage. Because container metadata don't include any port mapping information,
you must include service-specific port `dockerLabels` in your task definition container entries. A docker label of `ECS_TASK_OBSERVER_PORT` with a valid port
value will be attempted to be parsed for each reported container by default.

**An instance of the Collector must be running in the ECS task from which you want to detect containers.**

> :construction: This extension is in alpha and configuration fields are subject to change.

## Example Config

```yaml
extensions:
  ecs_task_observer:
    # the task metadata endpoint. If not set, detected by first of ECS_CONTAINER_METADATA_URI_V4 and ECS_CONTAINER_METADATA_URI
    # environment variables by default.
    endpoint: http://my.task.metadata.endpoint
    # the dockerLabels to use to try to extract target application ports. If not set "ECS_TASK_OBSERVER_PORT" will be used by default.
    port_labels: [A_DOCKER_LABEL_CONTAINING_DESIRED_PORT, ANOTHER_DOCKER_LABEL_CONTAINING_DESIRED_PORT]
    refresh_interval: 10s

receivers:
  receiver_creator:
    receivers:
      redis:
        rule: type == "container" && name matches "redis"
        config:
          password: `container.labels["SECRET"]`
    watch_observers: [ecs_task_observer]
```

The above config defines a custom task metadata endpoint and provides two port labels that will be used to set the resulting container endpoint's `port`.
A corresponding redis container definition could look like the following:

```json
{
  "containerDefinitions": [
    {
      "portMappings": [
        {
          "containerPort": 6379,
          "hostPort": 6379
        }
      ],
      "image": "redis",
      "dockerLabels": {
        "A_DOCKER_LABEL_CONTAINING_DESIRED_PORT": "6379",
        "SECRET": "my-redis-auth"
      },
      "name": "redis"
    }
  ]
}
```


### Config

As a rest client-utilizing extension, most of the ECS Task Observer's configuration is inherited from the Collector core
[HTTP Client Configuration Settings](https://github.com/open-telemetry/opentelemetry-collector/blob/main/config/confighttp/README.md#client-configuration).

All fields are optional.

| Name | Type | Default | Docs |
| ---- | ---- | ------- | ---- |
| endpoint |string| <no value> | The task metadata endpoint, detected from first of `ECS_CONTAINER_METADATA_URI_V4` and `ECS_CONTAINER_METADATA_URI` environment variables by default |
| tls |[configtls-TLSClientSetting](#configtls-tlsclientsetting)| <no value> | TLSSetting struct exposes TLS client configuration.  |
| read_buffer_size |int| <no value> | ReadBufferSize for HTTP client. See http.Transport.ReadBufferSize.  |
| write_buffer_size |int| <no value> | WriteBufferSize for HTTP client. See http.Transport.WriteBufferSize.  |
| timeout |[time-Duration](#time-duration)| <no value> | Timeout parameter configures `http.Client.Timeout`.  |
| headers |map[string]string| <no value> | Additional headers attached to each HTTP request sent by the client. Existing header values are overwritten if collision happens.  |
| auth |[Authentication]| <no value> | Auth configuration for outgoing HTTP calls.  |
| refresh_interval |[time-Duration](#time-duration)| 30s | RefreshInterval determines the frequency at which the observer needs to poll for collecting new information about task containers.  |
| port_labels |[]string| `[ECS_TASK_OBSERVER_PORT]` | PortLabels is a list of container Docker labels from which to obtain the observed Endpoint port. The first label with valid port found will be used.  If no PortLabels provided, default of ECS_TASK_OBSERVER_PORT will be used.  |

### configtls-TLSClientSetting

| Name | Type | Default | Docs |
| ---- | ---- | ------- | ---- |
| ca_file |string| <no value> | Path to the CA cert. For a client this verifies the server certificate. For a server this verifies client certificates. If empty uses system root CA. (optional)  |
| cert_file |string| <no value> | Path to the TLS cert to use for TLS required connections. (optional)  |
| key_file |string| <no value> | Path to the TLS key to use for TLS required connections. (optional)  |
| min_version |string| <no value> | MinVersion sets the minimum TLS version that is acceptable. If not set, TLS 1.0 is used. (optional)  |
| max_version |string| <no value> | MaxVersion sets the maximum TLS version that is acceptable. If not set, TLS 1.3 is used. (optional)  |
| insecure |bool| <no value> | In gRPC when set to true, this is used to disable the client transport security. See https://godoc.org/google.golang.org/grpc#WithInsecure. In HTTP, this disables verifying the server's certificate chain and host name (InsecureSkipVerify in the tls Config). Please refer to https://godoc.org/crypto/tls#Config for more information. (optional, default false)  |
| insecure_skip_verify |bool| <no value> | InsecureSkipVerify will enable TLS but not verify the certificate.  |
| server_name_override |string| <no value> | ServerName requested by client for virtual hosting. This sets the ServerName in the TLSConfig. Please refer to https://godoc.org/crypto/tls#Config for more information. (optional)  |

### time-Duration
An optionally signed sequence of decimal numbers, each with a unit suffix, such as `300ms`, `-1.5h`, or `2h45m`. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`.

[Authentication]: https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configauth