# Jaeger Exporter

Exports trace data to [Jaeger](https://www.jaegertracing.io/) collectors.

The following settings are required:

- `endpoint` (no default): host:port to which the exporter is going to send Jaeger trace data,
using the gRPC protocol. The valid syntax is described at
https://github.com/grpc/grpc/blob/master/doc/naming.md

The following settings can be optionally configured:

- `insecure` (default = false): whether to enable client transport security for
  the exporter's gRPC connection. See
  [grpc.WithInsecure()](https://godoc.org/google.golang.org/grpc#WithInsecure).
- `ca_file` path to the CA cert. For a client this verifies the server certificate. Should
  only be used if `insecure` is set to true.
- `cert_file` path to the TLS cert to use for TLS required connections. Should
  only be used if `insecure` is set to true.
- `key_file` path to the TLS key to use for TLS required connections. Should
  only be used if `insecure` is set to true.
- `keepalive` keepalive parameters for client gRPC. See
[grpc.WithKeepaliveParams()](https://godoc.org/google.golang.org/grpc#WithKeepaliveParams).
- `server_name_override` If set to a non-empty string, it will override the virtual host name 
of authority (e.g. :authority header field) in requests (typically used for testing).
- `balancer_name`(default = pick_first): Sets the balancer in grpclb_policy to discover the servers.
See [grpc loadbalancing example](https://github.com/grpc/grpc-go/blob/master/examples/features/load_balancing/README.md).
- `timeout` (default = 5s): Is the timeout for every attempt to send data to the backend.
- `retry_on_failure`
  - `enabled` (default = true)
  - `initial_interval` (default = 5s): Time to wait after the first failure before retrying; ignored if `enabled` is `false`
  - `max_interval` (default = 30s): Is the upper bound on backoff; ignored if `enabled` is `false`
  - `max_elapsed_time` (default = 120s): Is the maximum amount of time spent trying to send a batch; ignored if `enabled` is `false`
- `sending_queue`
  - `enabled` (default = false)
  - `num_consumers` (default = 10): Number of consumers that dequeue batches; ignored if `enabled` is `false`
  - `queue_size` (default = 5000): Maximum number of batches kept in memory before data; ignored if `disabled` is `false`;
  User should calculate this as `num_seconds * requests_per_second` where:
    - `num_seconds` is the number of seconds to buffer in case of a backend outage
    - `requests_per_second` is the average number of requests per seconds.

Example:

```yaml
exporters:
  jaeger:
    endpoint: jaeger-all-in-one:14250
    cert_pem_file: /my-cert.pem
    server_name_override: opentelemetry.io
```

The full list of settings exposed for this exporter are documented [here](./config.go)
with detailed sample configurations [here](./testdata/config.yaml).
