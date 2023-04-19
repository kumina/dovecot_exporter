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
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/kingpin/v2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var dovecotUpDesc = prometheus.NewDesc(
	prometheus.BuildFQName("dovecot", "", "up"),
	"Whether scraping Dovecot's metrics was successful.",
	[]string{"scope"},
	nil)

// CollectFromReader converts the output of Dovecot's EXPORT command to metrics.
func CollectFromReader(file io.Reader, scope string, ch chan<- prometheus.Metric) error {
	if scope == "global" {
		return collectGlobalMetricsFromReader(file, scope, ch)
	}
	return collectDetailMetricsFromReader(file, scope, ch)
}

// CollectFromFile collects dovecot statistics from the given file
func CollectFromFile(path string, scope string, ch chan<- prometheus.Metric) error {
	conn, err := os.Open(path)
	if err != nil {
		return err
	}
	return CollectFromReader(conn, scope, ch)
}

// CollectFromSocket collects statistics from dovecot's stats socket.
func CollectFromSocket(path string, scope string, ch chan<- prometheus.Metric) error {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return err
	}
	_, err = conn.Write([]byte("EXPORT\t" + scope + "\n"))
	if err != nil {
		return err
	}
	return CollectFromReader(conn, scope, ch)
}

// collectGlobalMetricsFromReader collects dovecot "global" scope metrics from
// the supplied reader.
func collectGlobalMetricsFromReader(reader io.Reader, scope string, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	// Read first line of input, containing the aggregation and column names.
	if !scanner.Scan() {
		return fmt.Errorf("Failed to extract columns from input")
	}
	columnNames := strings.Fields(scanner.Text())
	if len(columnNames) < 1 {
		return fmt.Errorf("Input does not provide any columns")
	}

	columns := []*prometheus.Desc{}
	for _, columnName := range columnNames {
		columns = append(columns, prometheus.NewDesc(
			prometheus.BuildFQName("dovecot", scope, columnName),
			"Help text not provided by this exporter.",
			[]string{},
			nil))
	}

	// Global metrics only have a single row containing values following the
	// line with column names
	if !scanner.Scan() {
		return scanner.Err()
	}
	values := strings.Fields(scanner.Text())

	if len(values) != len(columns) {
		return fmt.Errorf("error while parsing row: value count does not match column count")
	}

	for i, value := range values {
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		ch <- prometheus.MustNewConstMetric(
			columns[i],
			prometheus.UntypedValue,
			f,
		)
	}
	return scanner.Err()
}

// collectGlobalMetricsFromReader collects dovecot "non-global" scope metrics
// from the supplied reader.
func collectDetailMetricsFromReader(reader io.Reader, scope string, ch chan<- prometheus.Metric) error {
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	// Read first line of input, containing the aggregation and column names.
	if !scanner.Scan() {
		return fmt.Errorf("Failed to extract columns from input")
	}
	columnNames := strings.Split(scanner.Text(), "\t")
	if len(columnNames) < 2 {
		return fmt.Errorf("Input does not provide any columns")
	}

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
		row := scanner.Text()
		if strings.TrimSpace(row) == "" {
			break
		}

		values := strings.Split(row, "\t")
		if len(values) != len(columns)+1 {
			return fmt.Errorf("error while parsing rows: value count does not match column count")
		}
		if values[0] == "" {
			values[0] = "empty_user"
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

type DovecotExporter struct {
	scopes     []string
	socketPath string
}

func NewDovecotExporter(socketPath string, scopes []string) *DovecotExporter {
	return &DovecotExporter{
		scopes:     scopes,
		socketPath: socketPath,
	}
}

func (e *DovecotExporter) Describe(ch chan<- *prometheus.Desc) {
	ch <- dovecotUpDesc
}

func (e *DovecotExporter) Collect(ch chan<- prometheus.Metric) {
	for _, scope := range e.scopes {
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
		app           = kingpin.New("dovecot_exporter", "Prometheus metrics exporter for Dovecot")
		listenAddress = app.Flag("web.listen-address", "Address to listen on for web interface and telemetry.").Default(":9166").String()
		metricsPath   = app.Flag("web.telemetry-path", "Path under which to expose metrics.").Default("/metrics").String()
		socketPath    = app.Flag("dovecot.socket-path", "Path under which to expose metrics.").Default("/var/run/dovecot/stats").String()
		dovecotScopes = app.Flag("dovecot.scopes", "Stats scopes to query (comma separated)").Default("user").String()
	)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	exporter := NewDovecotExporter(*socketPath, strings.Split(*dovecotScopes, ","))
	prometheus.MustRegister(exporter)

	http.Handle(*metricsPath, promhttp.Handler())
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
