// prometheus-expvar-exporter collects expvar metrics from different sources,
// and exports them for Prometheus.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/pelletier/go-toml"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	configPath = flag.String("config", "config.toml",
		"configuration file")
)

func main() {
	flag.Parse()

	config, err := toml.LoadFile(*configPath)
	if err != nil {
		log.Fatalf("error loading config file %q: %v", *configPath, err)
	}

	for _, t := range config.Keys() {
		if !config.Has(t + ".url") {
			continue
		}

		c := &Collector{
			url:       config.Get(t + ".url").(string),
			names:     map[string]string{},
			helps:     map[string]string{},
			labelName: map[string]string{},
		}

		mnames := config.GetDefault(t+".m", &toml.Tree{}).(*toml.Tree).Keys()
		for _, name := range mnames {
			info := config.Get(t + ".m." + name).(*toml.Tree)
			expvar := info.Get("expvar").(string)
			c.names[expvar] = name
			if info.Has("help") {
				c.helps[expvar] = info.Get("help").(string)
			}
			if info.Has("label_name") {
				c.labelName[expvar] = info.Get("label_name").(string)
			}
		}

		log.Printf("Collecting %q\n", c.url)
		prometheus.MustRegister(c)
	}

	http.HandleFunc("/", indexHandler)
	http.Handle("/metrics", promhttp.Handler())

	if !config.Has("listen_addr") {
		log.Fatal("Configuration has no listen_addr")
	}
	addr := config.Get("listen_addr").(string)
	log.Printf("Listening on %q", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

type Collector struct {
	// URL to collect from.
	url string

	// expvar -> prometheus name
	names map[string]string

	// expvar -> prometheus help
	helps map[string]string

	// expvar -> prometheus label name
	labelName map[string]string
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	// Not returning anything is explicitly allowed, and seems to fit our use
	// case.
	// From the documentation:
	//   Sending no descriptor at all marks the Collector as “unchecked”, i.e.
	//   no checks will be performed at registration time, and the Collector
	//   may yield any Metric it sees fit in its Collect method/
	return
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	resp, err := http.Get(c.url)
	if err != nil {
		log.Printf("Error scraping %q: %v", c.url, err)
		return
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading body of %q: %v", c.url, err)
		return
	}

	// Replace "\xNN" with "?" because the default parser doesn't handle them
	// well.
	re := regexp.MustCompile(`\\x..`)
	body = re.ReplaceAllFunc(body, func(s []byte) []byte {
		return []byte("?")
	})

	var vs map[string]interface{}
	err = json.Unmarshal(body, &vs)
	if err != nil {
		log.Printf("Error unmarshalling json from %q: %v", c.url, err)
		return
	}

	for k, v := range vs {
		name := sanitizeMetricName(k)
		if n, ok := c.names[k]; ok {
			name = n
		}

		help := fmt.Sprintf("expvar %q", k)
		if h, ok := c.helps[k]; ok {
			help = h
		}

		lnames := []string{}
		if ln, ok := c.labelName[k]; ok {
			lnames = append(lnames, ln)
		}

		desc := prometheus.NewDesc(name, help, lnames, nil)

		switch v := v.(type) {
		case float64:
			ch <- prometheus.MustNewConstMetric(desc, prometheus.UntypedValue, v)
		case bool:
			ch <- prometheus.MustNewConstMetric(desc, prometheus.UntypedValue,
				valToFloat(v))
		case map[string]interface{}:
			// We only support explicitly written label names.
			if len(lnames) != 1 {
				continue
			}
			for lk, lv := range v {
				ch <- prometheus.MustNewConstMetric(desc, prometheus.UntypedValue,
					valToFloat(lv), lk)
			}
		case string:
			// Not supported by Prometheus.
			continue
		case []interface{}:
			// Not supported by Prometheus.
			continue
		default:
			// TODO: support nested labels / richer structures?
			//fmt.Printf("Not supported: %q %#v\n", name, v)
			continue
		}
	}
}

func valToFloat(v interface{}) float64 {
	switch v := v.(type) {
	case float64:
		return v
	case bool:
		if v {
			return 1.0
		}
		return 0.0
	}
	panic(fmt.Sprintf("unexpected value type: %#v", v))
}

func sanitizeMetricName(n string) string {
	// Prometheus metric names must match the regex
	// `[a-zA-Z_:][a-zA-Z0-9_:]*`.
	// https://prometheus.io/docs/concepts/data_model/#metric-names-and-labels
	//
	// This function replaces all non-matching ASCII characters with
	// underscores.
	//
	// In particular, it is common that expvar names contain `/` or `-`, which
	// we replace with `_` so they end up resembling more Prometheus-ideomatic
	// names.
	//
	// Non-ascii characters are not supported, and will panic as so to force
	// users to handle them explicitly.  There is no good way to handle all of
	// them automatically, as they can't be all reasonably mapped to ascii. In
	// the future, we may handle _some_ of them automatically when possible.
	// But for now, forcing the users to be explicit is the safest option, and
	// also ensures forwards compatibility.
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r
		}
		if r >= 'A' && r <= 'Z' {
			return r
		}
		if r >= '0' && r <= '9' {
			return r
		}
		if r == '_' || r == ':' {
			return r
		}
		if r > unicode.MaxASCII {
			panic(fmt.Sprintf(
				"non-ascii character %q is unsupported, please configure the metric %q explicitly",
				r, n))
		}
		return '_'
	}, n)
}

const indexHTML = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <title>expvar exporter</title>
  </head>
  <body>
    <h1>Prometheus expvar exporter</h1>

    This is a <a href="https://prometheus.io">Prometheus</a>
    <a href="https://prometheus.io/docs/instrumenting/exporters/">exporter</a>,
    takes <a href="https://golang.org/pkg/expvar/">expvars</a> and converts
    them to Prometheus metrics.<p>

    Go to <tt><a href="/metrics">/metrics</a></tt> to see the exported metrics.

  </body>
</html
`

func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(indexHTML))
}
