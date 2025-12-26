---
title: Building Resilient Observability Pipelines (Part I)
date: 2025-12-26T02:38:00Z
---


## Understand the environment data being generated

To build a resilient data pipeline, you first need to know where your data comes from. This information helps you decide how many collectors you need, if you need clustering, and how to configure your receivers.

I won't go into deep architectural details here. Instead, let's focus on the configuration.

## The First Line of Defense: Tuning the OTLP Receiver for High Scale

In this post, I will focus on Grafana Alloy. However, Alloy uses upstream components from the OpenTelemetry Collector. This means you can apply these settings 1:1 to the standard OTel Collector too.

OpenTelemetry provides a "Collector" to receive telemetry signals: metrics, logs, and traces. It is called a pipeline because you pipe one component to another to build a data flow.

You might ask: What can go wrong with that? Well, many things. :)

We will go through the pipeline component by component. We will look for a setup that is failure-tolerant and performant. This post is the first in a series, starting with the entry point: the Receiver.

## Deep Dive: OTLP Receiver

The OTLP receiver (`otelcol.receiver.otlp`) is the entry point for telemetry data into your collector. It receives data via gRPC or HTTP using the [OTLP](https://opentelemetry.io/docs/specs/otlp/) format. Understanding how to configure this component properly is crucial for building a resilient pipeline, as it's the first line of defense against overload.

### Basic Configuration

Here is a production-grade configuration for Alloy:

```hcl
otelcol.receiver.otlp "default" {
    grpc {
        endpoint = "0.0.0.0:4317"
        max_recv_msg_size = "4MiB"
        read_buffer_size = "512KiB"
        write_buffer_size = "32KiB"
        
        
        
        keepalive {
            server_parameters {
                max_connection_age = "2h"
                max_connection_age_grace = "10s"
            }
        }
    }

    http {
        endpoint = "0.0.0.0:4318"
        include_metadata = false
        max_request_body_size = "20MiB"

    }

    debug_metrics {
        disable_high_cardinality_metrics = true
    }
    output {
        metrics = [otelcol.<next_component>.default.input]
        logs = [otelcol.<next_component>.default.input]
        traces = [otelcol.<next_component>.default.input]
    }
}
```

### Why These Configuration Options are Matter?

#### gRPC Configuration

`endpoint` This is the network address the server listens on. Using `0.0.0.0` allows connections from any network interface. This is standard for containerized environments (like Docker or Kubernetes).

- Note on Localhost: If you set the endpoint to 127.0.0.1, you might see error logs about IPv6 (e.g., dial tcp [::1]:4317... connection refused). This is normal. The collector tries IPv6 first, fails, and then successfully connects on IPv4.

`max_recv_msg_size` This sets the maximum size of a single gRPC message. The default is 4MiB. This is usually enough, but heavy traces (like Java stack traces) can be larger.

*Warning*: Be careful when increasing this. If you set it to 100MiB and you have 100 simultaneous connections, your memory usage could spike by 10GB. This will likely crash your collector (OOMKill).

`read_buffer_size` The default (`512KiB`) is a good balance between memory usage and speed. It helps handle small bursts of network traffic.

`write_buffer_size` This controls the buffer for sending data back to the client. Since the receiver mostly just sends small "Acknowledgements" (ACKs), `32KiB` is enough.

`keepalive.server_parameters` These settings are critical if you use a Load Balancer:

- `max_connection_age`: This forces the client to reconnect after `2 hours`. This prevents one collector from holding onto all the connections forever. It ensures traffic is balanced evenly across all your collectors.
- `max_connection_age_grace`: A grace period (e.g., 10s) that allows in-flight requests to complete before the connection is severed.

#### HTTP Configuration

`endpoint` Port `4318` is the standard port for OTLP over HTTP.

`max_request_body_size` We set this to 20MiB. HTTP requests include JSON overhead, so they are often larger than gRPC messages. This allows for larger batches of data.

#### Resilience Considerations

- `Connection Management`: The keepalive settings ensure connections rotate. This prevents "stale" connections that eat up resources.

- `Message Size Limits`: Both `max_recv_msg_size` and `max_request_body_size` act as safety guards. They stop huge payloads from crashing your memory.

- `Buffer Sizing`: Proper buffers prevent data loss during small network spikes.

- `Dual Protocols`: Supporting both gRPC and `HTTP` is good practice. Use `gRPC` for services (it is faster/efficient) and `HTTP` for web clients or serverless functions.

#### Performance Tuning Tips

- High Throughput: If you see dropped connections, try increasing `read_buffer_size`.

- Low Latency: If you scale your collectors up and down often, reduce `max_connection_age` to force clients to reconnect to new collectors faster.

- Low Memory: If you are short on RAM, decrease the buffer sizes and message limits.

#### Common Pitfalls

Setting message sizes too high: This creates a risk of memory crashes during traffic spikes.

Ignoring keepalive: This can lead to uneven load balancing.

Binding to localhost only: This prevents other containers from reaching your collector.

Enabling high cardinality metrics: Always keep `disable_high_cardinality_metrics` = true in production, or your metrics bill will explode.


#### Security Notes

1. You can enable authentication for the grpc and http endpoints.

```hcl
otelcol.receiver.otlp "default" {
  http {
    auth = otelcol.auth.basic.creds.handler
  }
  grpc {
     auth = otelcol.auth.basic.creds.handler
  }
}

otelcol.auth.basic "creds" {
    username = sys.env("<USERNAME>")
    password = sys.env("<PASSWORD>")
}
```
2. TLS can be configured as needed. Please see [details](https://grafana.com/docs/alloy/latest/reference/components/otelcol/otelcol.receiver.otlp/#tls).

Overall, constantly revisit your configuration over time and tweak it as needed.

#### What's Next?

Configuring the receiver is just step one. Now that we are accepting data, we need to make sure we don't crash while processing it.

In the next post, we will look at `otelcol.processor.memory_limiter`.

