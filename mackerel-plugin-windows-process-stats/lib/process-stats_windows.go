//go:build windows

package mpwindowsprocessstats

import (
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"

	mp "github.com/mackerelio/go-mackerel-plugin-helper"
	"github.com/mackerelio/golib/logging"
	"github.com/yusufpapurcu/wmi"
)

var logger = logging.GetLogger("metrics.plugin.windows-process-stats")

type Win32_PerfFormattedData_PerfProc_Process struct {
	ElapsedTime          uint64
	Name                 string
	IDProcess            uint32
	PercentProcessorTime uint64
	WorkingSet           uint64
}

type WindowsProcessStatsPlugin struct {
	Process string
	Prefix  string
}

func getProcesses(processName string) ([]Win32_PerfFormattedData_PerfProc_Process, error) {
	var procs []Win32_PerfFormattedData_PerfProc_Process

	q := wmi.CreateQuery(&procs, "WHERE (Name like '"+processName+"' OR Name like '"+processName+"#%')")
	if err := wmi.Query(q, &procs); err != nil {
		return procs, err
	}

	sort.Slice(procs, func(i, j int) bool {
		return procs[i].IDProcess < procs[j].IDProcess
	})
	return procs, nil
}

// FetchMetrics interface for mackerelplugin
func (m WindowsProcessStatsPlugin) FetchMetrics() (map[string]interface{}, error) {
	procs, err := getProcesses(m.Process)
	if err != nil {
		return nil, err
	}
	stat := make(map[string]interface{})
	var re = regexp.MustCompile(`#[0-9]+$`)
	for k, v := range procs {
		processName := re.ReplaceAllString(v.Name, "") + "_" + strconv.Itoa(k)
		metricNameCPU := "cpu." + processName + ".percent_processor_time"
		metricNameMemory := "memory." + processName + ".working_set"
		stat[metricNameCPU] = v.PercentProcessorTime
		stat[metricNameMemory] = v.WorkingSet
	}

	return stat, nil
}

// GraphDefinition interface for mackerelplugin
func (m WindowsProcessStatsPlugin) GraphDefinition() map[string](mp.Graphs) {
	prefix := m.Prefix
	return map[string](mp.Graphs){
		"cpu.#": mp.Graphs{
			Label: fmt.Sprintf("%s Windows Process Stats CPU", prefix),
			Unit:  "percentage",
			Metrics: []mp.Metrics{
				{Name: "percent_processor_time", Label: "cpu", Diff: false, Stacked: false},
			},
		},
		"memory.#": mp.Graphs{
			Label: fmt.Sprintf("%s Windows Process Stats Memory", prefix),
			Unit:  "bytes",
			Metrics: []mp.Metrics{
				{Name: "working_set", Label: "memory", Diff: false, Stacked: false},
			},
		},
	}
}

// MetricKeyPrefix interface for mackerelplugin
func (m WindowsProcessStatsPlugin) MetricKeyPrefix() string {
	if m.Prefix == "" {
		return "windows-process-stats-" + m.Process
	}
	return m.Prefix
}

// Do the plugin
func Do() {
	optProcess := flag.String("process", "", "Process name")
	optMetricKeyPrefix := flag.String("metric-key-prefix", "", "Metric Key Prefix")
	optTempfile := flag.String("tempfile", "", "Temp file name")
	flag.Parse()

	if *optProcess == "" {
		logger.Warningf("Process name is required")
		flag.PrintDefaults()
		os.Exit(1)
	}

	var plugin WindowsProcessStatsPlugin
	plugin.Process = *optProcess
	plugin.Prefix = *optMetricKeyPrefix

	helper := mp.NewMackerelPlugin(plugin)
	helper.Tempfile = *optTempfile
	helper.Run()
}
