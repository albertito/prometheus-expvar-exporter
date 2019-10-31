
# prometheus-expvar-exporter

[prometheus-expvar-exporter] collects [expvar] metrics from different sources,
and [exports] them for [Prometheus].

[prometheus-expvar-exporter]: https://blitiri.com.ar/git/r/prometheus-expvar-exporter/
[expvar]: https://golang.org/pkg/expvar/
[exports]: https://prometheus.io/docs/instrumenting/exporters/
[Prometheus]: https://prometheus.io/


## Install

```
go get blitiri.com.ar/go/prometheus-expvar-exporter
```


## Configure

The exporter is configured using a [toml](https://en.wikipedia.org/wiki/TOML)
file.

For example:

```toml
# Address to listen on. Prometheus should be told to scrape this.
listen_addr = ":8000"

# Expvar target.
[server]
url = "http://localhost:2222/debug/vars"
```

By default, the exporter will try to auto-convert all expvars.  It is only
able to auto-convert float and boolean expvars without labels. The
configuration allows for manual set up of conversion, and label support.

See [config.toml](config.toml) for more details and an example.


## Run

```
~/go/bin/prometheus-expvar-exporter --config=config.toml
```
