// Copyright 2018 The aquachain Authors
// This file is part of the aquachain library.
//
// The aquachain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The aquachain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the aquachain library. If not, see <http://www.gnu.org/licenses/>.

//go:build !windows && !solaris && !nacl
// +build !windows,!solaris,!nacl

package metrics

import (
	"fmt"
	"log/syslog"
	"time"
)

// Output each metric in the given registry to syslog periodically using
// the given syslogger.
func Syslog(r Registry, d time.Duration, w *syslog.Writer) {
	for range time.Tick(d) {
		r.Each(func(name string, i interface{}) {
			switch metric := i.(type) {
			case Counter:
				w.Info(fmt.Sprintf("counter %s: count: %d", name, metric.Count()))
			case Gauge:
				w.Info(fmt.Sprintf("gauge %s: value: %d", name, metric.Value()))
			case GaugeFloat64:
				w.Info(fmt.Sprintf("gauge %s: value: %f", name, metric.Value()))
			case Healthcheck:
				metric.Check()
				w.Info(fmt.Sprintf("healthcheck %s: error: %v", name, metric.Error()))
			case Histogram:
				h := metric.Snapshot()
				ps := h.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				w.Info(fmt.Sprintf(
					"histogram %s: count: %d min: %d max: %d mean: %.2f stddev: %.2f median: %.2f 75%%: %.2f 95%%: %.2f 99%%: %.2f 99.9%%: %.2f",
					name,
					h.Count(),
					h.Min(),
					h.Max(),
					h.Mean(),
					h.StdDev(),
					ps[0],
					ps[1],
					ps[2],
					ps[3],
					ps[4],
				))
			case Meter:
				m := metric.Snapshot()
				w.Info(fmt.Sprintf(
					"meter %s: count: %d 1-min: %.2f 5-min: %.2f 15-min: %.2f mean: %.2f",
					name,
					m.Count(),
					m.Rate1(),
					m.Rate5(),
					m.Rate15(),
					m.RateMean(),
				))
			case Timer:
				t := metric.Snapshot()
				ps := t.Percentiles([]float64{0.5, 0.75, 0.95, 0.99, 0.999})
				w.Info(fmt.Sprintf(
					"timer %s: count: %d min: %d max: %d mean: %.2f stddev: %.2f median: %.2f 75%%: %.2f 95%%: %.2f 99%%: %.2f 99.9%%: %.2f 1-min: %.2f 5-min: %.2f 15-min: %.2f mean-rate: %.2f",
					name,
					t.Count(),
					t.Min(),
					t.Max(),
					t.Mean(),
					t.StdDev(),
					ps[0],
					ps[1],
					ps[2],
					ps[3],
					ps[4],
					t.Rate1(),
					t.Rate5(),
					t.Rate15(),
					t.RateMean(),
				))
			}
		})
	}
}
