---
title: Building Resilient Observability Pipelines (Part III)
date: 2026-06-07T10:00:00Z
---

---

## One collector, two pipelines, one memory

In [Part II](https://robustinfra.de/post/memory_limiter) we added the
`otelcol.processor.memory_limiter` to protect the collector from running out of
memory. That post used the percentage settings (`limit_percentage = 80`). I need
to correct that advice, and show a second, bigger problem.

You can run two pipelines in one Grafana Alloy collector. One pipeline takes
OpenTelemetry (OTLP) data. The other scrapes Prometheus metrics. This looks
nice: one deployment, one set of credentials, one thing to run.

But there is a hidden risk. The memory limiter protects only the OTLP pipeline.
It does nothing for the Prometheus pipeline. And both pipelines use the same
memory. So when the Prometheus side uses too much memory, it either starves the
healthy OTLP pipeline or kills the whole collector.

This post shows both problems on a small kind cluster that sends data to Grafana
Cloud. Then it shows how to fix the Prometheus side.

## First, fix the percentage setting

Part II suggested `limit_percentage = 80`. There is a trap here. The percentage
setting looks at the total memory of the host or node. It does not look at the
container memory limit.

Inside a 256 MiB pod on a large node, "80 percent" means several gigabytes. So
Kubernetes kills the pod long before the limiter starts. In a container, always
use fixed sizes, set below the container limit:

```alloy
otelcol.processor.memory_limiter "guard" {
  check_interval = "1s"
  limit          = "180MiB"   // hard limit, below the 256Mi container limit
  spike_limit    = "40MiB"    // soft limit = 180 - 40 = 140MiB
  output { ... }
}
```

## What memory_limiter can and cannot reach

Every second, the limiter checks the memory the process uses. If memory is too
high, it starts to refuse new data. "Refuse" means it returns an error to the
OTLP receiver in front of it, which then tells the sender to slow down. So the
limiter has only one tool: it pushes back on data that goes through the
OpenTelemetry pipeline.

The Prometheus pipeline is separate: `prometheus.scrape` collects samples,
`prometheus.relabel` changes them, and `prometheus.remote_write` sends them out.
This data never goes through the OpenTelemetry pipeline. So the limiter cannot
refuse, drop, or slow it. There is no memory limiter on the Prometheus side.

But both pipelines run in one process and share one memory (one Go heap). Every
series the scraper loads, and every batch the remote-write queue holds, uses the
same memory the limiter measures. So the limiter sees the pressure, but it
cannot fix the cause. It can only slow the OTLP pipeline. The pipeline that
caused the problem keeps running. The pipeline that did nothing wrong gets
punished.

## Reproducing it

The test is a kind cluster with one Alloy deployment that runs both pipelines.
It uses two load generators: `telemetrygen` sends a steady, healthy OTLP stream,
and `avalanche` is a Prometheus cardinality bomb that we can tune. Both pipelines
send to Grafana Cloud. Alloy also scrapes its own metrics, so we can watch the
failure on a dashboard.

There are two scenarios. They differ only in how hard we push the Prometheus
path and how much memory the container has.

## Problem one: the fast OOM (limiter is skipped)

Container limit: 256 MiB. avalanche serves about 400,000 series.

On each scrape, Alloy must load and parse all those series into memory at once.
This pushes the heap to about 410 MiB in less than one second. That is faster
than the limiter's one-second check. So Kubernetes sees the container go over
its limit and kills it (exit code 137) before the limiter even runs. The pod
keeps crashing and restarting.

```text
lastTerminatedReason = OOMKilled   exitCode = 137   restarts keep rising
otelcol_receiver_refused_metric_points_total = 0   (the limiter never ran)
```

Here the limiter is not protecting the wrong pipeline. It is skipped completely.
And it gets worse: when the collector dies, it also stops sending its own
metrics. So the metrics you need to understand the problem disappear. A
collector that runs out of memory goes blind while it falls.

![Fast OOM: the limiter is skipped and the collector keeps crashing](https://raw.githubusercontent.com/mbaykara/mbaykara.github.io/main/images/s2-blunt-oom.png)

## Problem two: the slow cascade (limiter starves the healthy pipeline)

Now give the container more room (1 GiB), so the limiter, not Kubernetes, is the
real limit. Use a steady load of about 40,000 series.

The heap now rises and stays between about 256 and 900 MiB. That is well above
the 180 MiB hard limit. So the limiter does its job: it refuses OTLP data. On
the dashboard, `otelcol_receiver_refused_metric_points_total` goes up, and
`otelcol_receiver_accepted_metric_points_total` drops to zero. The healthy
`telemetrygen` stream is now being dropped.

At the same time, the Prometheus pipeline that caused the problem is not slowed
at all. `prometheus.remote_write` keeps sending 6,000 to 7,000 samples per
second (about 250 kB/s) to the backend. So the pipeline at fault floods the
backend, while the healthy pipeline starves. And because the heap still spikes
near the memory limit, the collector also restarts a few times.

![Slow cascade: OTLP drops to zero while Prometheus floods the backend](https://raw.githubusercontent.com/mbaykara/mbaykara.github.io/main/images/s2-graded-cascade.png)

## How to protect the Prometheus path

The Prometheus pipeline has its own controls. None of them is a process-wide
memory limiter. But together they stop one noisy target from killing the
collector.

- **Limit the scrape.** `prometheus.scrape` supports `sample_limit`,
  `label_limit`, and `body_size_limit`. A `sample_limit` fails a scrape that has
  too many series, instead of loading all of them into memory.

```alloy
prometheus.scrape "targets" {
  sample_limit    = 50000
  label_limit     = 30
  body_size_limit = "10MiB"
}
```

- **Limit the remote-write queue.** `queue_config` (capacity, max shards)
  controls how much remote-write keeps in memory when the backend is slow.

```alloy
prometheus.remote_write "cloud" {
  endpoint {
    url = "..."
    queue_config {
      capacity   = 2500
      max_shards = 50
    }
  }
}
```

- **Use Adaptive Metrics.** For series you do not need, drop them with Grafana
  Cloud Adaptive Metrics instead of carrying them through the collector.

- **Split the collectors.** The safest option is to run the noisy Prometheus
  scraping in its own collector, with its own memory budget. Then a cardinality
  spike there cannot starve or kill the OTLP pipeline that shares its memory
  today.

## Summary

`otelcol.processor.memory_limiter` protects one pipeline, not the whole process.
In a mixed collector it can only slow the OpenTelemetry side, even when the
Prometheus side is the real cause. And with a small container limit, it does not
even get the chance to run.

If you run both pipelines in one Alloy collector, give the Prometheus path its
own limits, or give it its own collector. Do not expect the memory limiter to
cover the Prometheus side. It never did.
