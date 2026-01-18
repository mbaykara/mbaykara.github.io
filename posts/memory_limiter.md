---
title: Building Resilient Observability Pipelines (Part II)
date: 2026-01-18T14:41:00Z
---

---

## The Memory Limiter

In our previous [post](https://robustinfra.de/post/receivers), we set up the front door of your observability pipeline using `otelcol.receiver.otlp`. Today, we are moving down the pipeline to the next and one of the most critical aspects of running a production-grade collector: Resilience.

Specifically, we are deep diving into the [Memory Limiter Processor](https://github.com/open-telemetry/opentelemetry-collector/blob/main/processor/memorylimiterprocessor/README.md#memory-limiter-processor) (`otelcol.processor.memory_limiter`).

## The Well-Known `OOMKilled`

If you manage Kubernetes workloads, you are likely familiar with the `OOMKilled` status.

A process tries to consume more memory than the limit you defined in your deployment. The Linux kernel steps in like a bouncer, sends a `SIGKILL` signal, and terminates the process immediately to protect the rest of the node. Once Kubelet notices the dead container and marks the Pod status as `OOMKilled`, this is what you see in the Pod status.

For an OpenTelemetry Collector or Grafana Alloy instance, this results in a data gap.

Collectors are high throughput systems. They ingest metrics, tail logs, and process traces.

When

- a spike in traffic occurs, perhaps due to a deployment triggering error logs
- a sudden surge in user activity
- high cardinality metrics suddenly appear

memory usage can skyrocket, and many other unexpected causes can contribute as well.

Without protection, your collector will crash exactly when you need it most: during an Incident.

To prevent this, we need a defense-in-depth strategy involving two layers: The Runtime and The Application.

## Layer 1: The Runtime Guard (`GOMEMLIMIT`)

Before we configure the processor, we must ensure the Go runtime, which powers both the OTEL Collector and Alloy, is aware of its constraints.

Historically, Go was unaware of container memory limits. It would allocate memory until the OS killed it. With Go 1.19+, we have [GOMEMLIMIT](https://pkg.go.dev/runtime).

By setting the environment variable `GOMEMLIMIT` to roughly 80 to 90 percent of your container memory limit, you force the Go Garbage Collector to run more aggressively as usage approaches that limit. This prevents many OOM kills by trading some CPU for memory safety.

## Layer 2: The Application Guard (`memory_limiter`)

`GOMEMLIMIT` improves how memory is reclaimed, but it cannot stop new data from coming in. If the inflow is too high, the GC will not keep up.

This is where `otelcol.processor.memory_limiter` comes in.

This component sits in your pipeline and acts as a circuit breaker. It monitors the process memory usage periodically. If usage crosses a soft limit, it starts dropping data and forces garbage collection to reduce memory usage.

> It is better to drop 10 percent of your traces during a spike than to crash the collector and lose 100 percent of your data while waiting for a restart.

### How it Works

The processor relies on two main calculations:

1. The Limit: The absolute memory ceiling the processor tries to respect.
2. The Spike Limit: A buffer that allows temporary surges between checks.

### Configuration Deep Dive

Below is an example configuration for Grafana Alloy.

```alloy
otelcol.processor.memory_limiter "main" {
  check_interval         = "1s"
  limit_percentage       = 80
  spike_limit_percentage = 25

  output {
    metrics = [otelcol.exporter.otlp.default.input]
    logs    = [otelcol.exporter.otlp.default.input]
    traces  = [otelcol.exporter.otlp.default.input]
  }
}
```

### The Parameters

`check_interval`

- How often the processor checks memory usage.
- `Recommendation`: 1 second. Memory can spike very fast in high throughput systems.

`limit_percentage`

- The soft limit. When memory usage reaches this percentage, the processor starts dropping data.
- `Recommendation`: 75 to 80 percent.

`spike_limit_percentage`

- The gap between the soft limit and a hard crash.
- `Recommendation`: 20 to 25 percent.


`The math`: limit percentage plus spike limit percentage should be close to 100 percent.

You can also use `limit_mb` and `spike_limit_mb` for fixed values. Percentages are usually better in Kubernetes. For static environments like `VMs` or `bare metal`, fixed values can be a better choice.

### Where to place it?

`tldr: As early as possible.`

The Memory Limiter should be placed early in the pipeline, usually right after the receivers.

If you place it at the end of the pipeline, you may already have used a lot of memory before dropping data. It is better to fail fast and fail cheap.

Receiver -> Memory Limiter -> Batch -> Other Processors -> Exporter

### Summary Checklist

To build a resilient pipeline:

- Set memory requests and limits for the Pod.
- Set `GOMEMLIMIT` to about 90 percent of the Pod limit.
- Configure `memory_limiter`:
- `check_interval` = “1s”
- `limit_percentage` = 80
- `spike_limit_percentage` = 25
- Monitor dropped data metrics like `otelcol_processor_refused_spans`.

By applying these steps, your collector will handle load better and avoid crashes.

#### What’s Next?

Configuring the receiver was the first step. Now we accept data in a more resilient way by using the memory limiter. Next, we will enrich the data by detecting the environment.

In the next post, we will look at `otelcol.processor.resourcedetection`.
