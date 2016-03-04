package monitor

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"

	log "github.com/elleFlorio/gru/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/elleFlorio/gru/autonomic/monitor/logreader"
	cfg "github.com/elleFlorio/gru/configuration"
	"github.com/elleFlorio/gru/enum"
	res "github.com/elleFlorio/gru/resources"
	"github.com/elleFlorio/gru/service"
	"github.com/elleFlorio/gru/storage"
	"github.com/elleFlorio/gru/utils"
)

var (
	monitorActive  bool
	gruStats       GruStats
	history        statsHistory
	c_stop         chan struct{}
	c_err          chan error
	ErrNoIndexById error = errors.New("No index for such Id")
)

func init() {
	gruStats = GruStats{
		Service:  make(map[string]ServiceStats),
		Instance: make(map[string]InstanceStats),
	}
	history = statsHistory{make(map[string]instanceHistory)}
}

func Run() GruStats {
	log.WithField("status", "init").Debugln("Gru Monitor")
	defer log.WithField("status", "done").Debugln("Gru Monitor")
	snapshot := GruStats{
		Service:  make(map[string]ServiceStats),
		Instance: make(map[string]InstanceStats),
	}

	computeServicesStats(&gruStats)
	computeSystemCpu(&gruStats)
	updateSystemInstances(&gruStats)
	makeSnapshot(&gruStats, &snapshot)
	err := saveStats(snapshot)
	if err != nil {
		log.WithField("err", "Stats data not saved").Errorln("Running monitor")
	}

	services := service.List()
	for _, name := range services {
		resetEventsStats(name, &gruStats)
		resetMetricStats(name, &gruStats)
	}

	displayStatsOfServices(snapshot)
	return snapshot
}

func computeServicesStats(stats *GruStats) {
	metrics := metric.Manager().GetMetrics()
	for name, _ := range gruStats.Service {
		updateRunningInstances(name, &gruStats, 25)
		computeServiceCpuPerc(name, &gruStats)
		computeServiceMemory(name, &gruStats)
		updateServiceMetrics(name, metrics[name], &gruStats)
	}
}

//FIXME need to check if all the windows are actually ready
func updateRunningInstances(name string, stats *GruStats, wsize int) {
	srv, _ := service.GetServiceByName(name)
	pending := srv.Instances.Pending

	for _, inst := range pending {
		if len(history.instance[inst].cpu.sysUsage.Slice()) >= wsize {
			addInstance(inst, name, "running", stats, &history)

			log.WithFields(log.Fields{
				"service": name,
				"id":      inst,
			}).Debugln("Promoted resource to running state")
		}
	}
}

func computeServiceCpuPerc(name string, stats *GruStats) {
	sum := 0.0
	avg := 0.0
	srvStats := stats.Service[name]
	srv, _ := service.GetServiceByName(name)

	if len(srv.Instances.Running) > 0 {
		for _, id := range srv.Instances.Running {
			instCpus := history.instance[id].cpu.totalUsage.Slice()
			sysCpus := history.instance[id].cpu.sysUsage.Slice()
			instCpuAvg := computeInstanceCpuPerc(instCpus, sysCpus)
			inst := stats.Instance[id]
			inst.Cpu = instCpuAvg
			stats.Instance[id] = inst
			sum += instCpuAvg

			log.WithFields(log.Fields{
				"instance": id,
				"CPUavg":   instCpuAvg,
			}).Debugln("Computed local instance average CPU")
		}

		avg = sum / float64(len(srv.Instances.Running))
	}

	srvStats.Cpu.Avg = avg
	srvStats.Cpu.Tot = sum
	stats.Service[name] = srvStats

	log.WithFields(log.Fields{
		"Service": name,
		"CPUavg":  avg,
		"CPUsum":  sum,
	}).Debugln("Computed local service CPU usage")
}

// Since linux compute the cpu usage in units of jiffies, it needs to be converted
// in % using the formula used in this function.
// Explaination: http://stackoverflow.com/questions/1420426/calculating-cpu-usage-of-a-process-in-linux
// TODO probably I just need the first and the last value...
// 2015/11/16 - corrected according to what the docker client does:
// https://github.com/docker/docker/blob/master/api/client/stats.go#L316
func computeInstanceCpuPerc(instCpus []float64, sysCpus []float64) float64 {
	sum := 0.0
	instNext := 0.0
	sysNext := 0.0
	instPrev := 0.0
	sysPrev := 0.0
	cpu := 0.0

	valid := 0
	nValues := int(math.Min(float64(len(instCpus)), float64(len(sysCpus))))

	for i := 1; i < nValues; i++ {
		instPrev = instCpus[i-1]
		sysPrev = sysCpus[i-1]
		instNext = instCpus[i]
		sysNext = sysCpus[i]
		instDelta := instNext - instPrev
		if instDelta > 0 {
			sysDelta := sysNext - sysPrev
			if sysDelta == 0 {
				cpu = 0
			} else {
				// "100 * cpu" should produce values in [0, 100]
				cpu = (instDelta / sysDelta) * float64(res.GetResources().CPU.Total)
			}
			sum += cpu
			valid++
		}
	}

	return math.Min(1.0, sum/float64(valid))
}

func computeServiceMemory(name string, stats *GruStats) {
	sum := 0.0
	avg := 0.0
	srv, _ := service.GetServiceByName(name)
	memLimit := srv.Docker.Memory
	srvStats := stats.Service[name]

	if len(srv.Instances.Running) > 0 {
		for _, id := range srv.Instances.Running {
			instMem := history.instance[id].mem.Slice()
			instMemPerc := computeInstaceMemPerc(instMem, memLimit)
			inst := stats.Instance[id]
			inst.Memory = instMemPerc
			stats.Instance[id] = inst
			sum += instMemPerc

			log.WithFields(log.Fields{
				"instance": id,
				"Memory":   instMemPerc,
			}).Debugln("Computed local instance Memory")
		}

		avg = sum / float64(len(srv.Instances.Running))
	}

	srvStats.Memory.Avg = avg
	// If the service does not have a memory limit,
	// the sum of the memory used by all the instances
	// can exceed 100%. In this case the total memory used
	// is limited by the system memory and the total is
	// virtually equal to the average
	if memLimit != "" {
		srvStats.Memory.Tot = sum
	} else {
		srvStats.Memory.Tot = avg
	}
	stats.Service[name] = srvStats

	log.WithFields(log.Fields{
		"Service":   name,
		"MemoryAvg": avg,
		"MemorySum": sum,
	}).Debugln("Computed local service memory usage")
}

func computeInstaceMemPerc(instMem []float64, limit string) float64 {
	var err error
	totalMemory := cfg.GetNode().Resources.TotalMemory
	sum := 0.0
	avg := 0.0
	limitBytes := totalMemory
	if limit != "" {
		limitBytes, err = utils.RAMInBytes(limit)
		if err != nil {
			log.WithField("err", err).Errorln("Error computing instance memory usage")
			limitBytes = totalMemory
		}
	}

	for _, m := range instMem {
		sum += m
	}
	avg = sum / float64(len(instMem))

	return avg / float64(limitBytes)
}

func updateServiceMetrics(name string, metrics map[string][]float64, stats *GruStats) {
	if len(metrics) == 0 {
		log.Debugln("No metrics to update for service ", name)
		return
	}

	srv := stats.Service[name]
	for metric, value := range metrics {
		switch metric {
		case "execution_time":
			srv.Metrics.ResponseTime = value
			log.WithField("execution_time", value).Debugln("Updated execution time of service ", name)
		case "response_time":
			//TODO
		default:
			log.WithField("metric", metric).Errorln("Cannot update undefined metric of service ", name)
		}
	}
	stats.Service[name] = srv
}

func computeSystemCpu(stats *GruStats) {
	sum := 0.0
	for _, value := range stats.Service {
		sum += value.Cpu.Tot
	}
	stats.System.Cpu = math.Min(1.0, sum)
}

func updateSystemInstances(stats *GruStats) {
	cfg.ClearNodeInstances()
	instances := cfg.GetNodeInstances()
	for name, _ := range stats.Service {
		srv, _ := service.GetServiceByName(name)
		instances.All = append(instances.All, srv.Instances.All...)
		instances.Pending = append(instances.Pending, srv.Instances.Pending...)
		instances.Running = append(instances.Running, srv.Instances.Running...)
		instances.Stopped = append(instances.Stopped, srv.Instances.Stopped...)
		instances.Paused = append(instances.Paused, srv.Instances.Paused...)
	}
}

//TODO maybe I can just compute historical data without make a deep copy
// since now I'm serializing the structure in a string of bytes...
// check this possibility.
func makeSnapshot(src *GruStats, dst *GruStats) {
	// Copy service stats
	for name, stats := range src.Service {
		srv_src := stats
		// Copy events (NEEDED?)
		events_src := srv_src.Events
		stop_dst := make([]string, len(events_src.Stop), len(events_src.Stop))
		start_dst := make([]string, len(events_src.Start), len(events_src.Start))
		copy(start_dst, events_src.Start)
		copy(stop_dst, events_src.Stop)
		events_dst := EventStats{
			start_dst,
			stop_dst,
		}

		// Copy cpu
		cpu_dst := CpuStats{stats.Cpu.Avg, stats.Cpu.Tot}

		// Copy memory
		mem_dst := MemoryStats{stats.Memory.Avg, stats.Memory.Tot}

		// Copy metrics
		metrics_src := srv_src.Metrics
		respTime_dst := make([]float64, len(metrics_src.ResponseTime), len(metrics_src.ResponseTime))
		copy(respTime_dst, metrics_src.ResponseTime)
		metrics_dst := MetricStats{respTime_dst}

		srv_dst := ServiceStats{ /*status_dst, */ events_dst, cpu_dst, mem_dst, metrics_dst}
		dst.Service[name] = srv_dst
	}

	//Copy instance stats
	for id, value := range src.Instance {
		inst_dst := InstanceStats{value.Cpu, value.Memory}
		dst.Instance[id] = inst_dst
	}

	dst.System.Cpu = src.System.Cpu
}

func saveStats(stats GruStats) error {
	data, err := convertStatsToData(stats)
	if err != nil {
		log.WithField("err", err).Debugln("Cannot convert stats to data")
		return err
	} else {
		storage.StoreLocalData(data, enum.STATS)
	}

	return nil
}

func convertStatsToData(stats GruStats) ([]byte, error) {
	data, err := json.Marshal(stats)
	if err != nil {
		log.WithField("err", err).Errorln("Error marshaling stats data")
		return nil, err
	}

	return data, nil
}

func resetEventsStats(srvName string, stats *GruStats) {
	srvStats := stats.Service[srvName]

	log.WithFields(log.Fields{
		"service": srvName,
		"start":   srvStats.Events.Start,
		"stop":    srvStats.Events.Stop,
	}).Debugln("Monitored events")

	srvStats.Events = EventStats{}
	stats.Service[srvName] = srvStats
}

func resetMetricStats(srvName string, stats *GruStats) {
	srvStats := stats.Service[srvName]
	srvStats.Metrics = MetricStats{}
	stats.Service[srvName] = srvStats
}

func displayStatsOfServices(stats GruStats) {
	for name, value := range stats.Service {
		srv, _ := service.GetServiceByName(name)
		log.WithFields(log.Fields{
			"pending:": len(srv.Instances.Pending),
			"running:": len(srv.Instances.Running),
			"stopped:": len(srv.Instances.Stopped),
			"paused:":  len(srv.Instances.Paused),
			"cpu avg":  fmt.Sprintf("%.2f", value.Cpu.Avg),
			"cpu tot":  fmt.Sprintf("%.2f", value.Cpu.Tot),
			"mem avg":  fmt.Sprintf("%.2f", value.Memory.Avg),
			"mem tot":  fmt.Sprintf("%.2f", value.Memory.Tot),
		}).Infoln("Stats computed: ", name)
	}
}

func GetMonitorData() (GruStats, error) {
	stats := GruStats{}
	dataStats, err := storage.GetLocalData(enum.STATS)
	if err != nil {
		log.WithField("err", err).Errorln("Cannot retrieve stats data.")
	} else {
		stats, err = convertDataToStats(dataStats)
	}

	return stats, err
}

func convertDataToStats(data []byte) (GruStats, error) {
	stats := GruStats{}
	err := json.Unmarshal(data, &stats)
	if err != nil {
		log.WithField("err", err).Errorln("Error unmarshaling stats data")
	}

	return stats, err
}
