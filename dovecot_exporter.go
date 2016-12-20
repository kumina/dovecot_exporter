// Copyright 2016 Kumina, https://kumina.nl/
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	dovecotUpDesc = prometheus.NewDesc(
		prometheus.BuildFQName("dovecot", "", "up"),
		"Whether scraping Dovecot's metrics was successful.",
		[]string{"scope"},
		nil)
	dovecotScopes = [...]string{"user"}
)

// Converts the output of Dovecot's EXPORT command to metrics.
func CollectFromReader(file io.Reader, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanLines)

	// Read first line of input, containing the aggregation and column names.
	if !scanner.Scan() {
		return fmt.Errorf("Failed to extract columns from input")
	}
	columnNames := strings.Fields(scanner.Text())
	columns := []*prometheus.Desc{}
	for _, columnName := range columnNames[1:] {
		columns = append(columns, prometheus.NewDesc(
			prometheus.BuildFQName("dovecot", columnNames[0], columnName),
			"Help text not provided by this exporter.",
			[]string{columnNames[0]},
			nil))
	}

	// Read successive lines, containing the values.
	for scanner.Scan() {
		values := strings.Fields(scanner.Text())
		if len(values) < 1 {
			break
		}
		for i, value := range values[1:] {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return err
			}
			ch <- prometheus.MustNewConstMetric(
				columns[i],
				prometheus.UntypedValue,
				f,
				values[0])
		}
	}
	return scanner.Err()
}

func CollectFromFile(path string, ch chan<- prometheus.Metric) error {
	conn, err := os.Open(path)
	if err != nil {
		return err
	}
	return CollectFromReader(conn, ch)
}

func CollectFromSocket(path string, scope string, ch chan<- prometheus.Metric) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte("EXPORT\t" + scope + "\n"))
	if err != nil {
		return err
	}
	return CollectFromReader(conn, ch)
}

type DovecotExporter struct {
	socketPath string
}

func NewDovecotExporter(socketPath string) *DovecotExporter {
	return &DovecotExporter{
		socketPath: socketPath,
	}
}

func (e *DovecotExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- dovecotUpDesc
}

func (e *DovecotExporter) Collect(ch chan<- prometheus.Metric) {
	for _, scope := range dovecotScopes {
		err := CollectFromSocket(e.socketPath, scope, ch)
		if err == nil {
			ch <- prometheus.MustNewConstMetric(
				dovecotUpDesc,
				prometheus.GaugeValue,
				1.0,
				scope)
		} else {
			log.Printf("Failed to scrape socket: %s", err)
			ch <- prometheus.MustNewConstMetric(
				dovecotUpDesc,
				prometheus.GaugeValue,
				0.0,
				scope)
		}
	}
}

func main() {
	var (
		listenAddress = flag.String("web.listen-address", ":9199", "Address to listen on for web interface and telemetry.")
		metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
		socketPath    = flag.String("dovecot.socket-path", "/var/run/dovecot/stats", "Path under which to expose metrics.")
	)
	flag.Parse()

	exporter := NewDovecotExporter(*socketPath)
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`
			<html>
			<head><title>Dovecot Exporter</title></head>
			<body>
			<h1>Dovecot Exporter</h1>
			<p><a href='` + *metricsPath + `'>Metrics</a></p>
			</body>
			</html>`))
	})
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
