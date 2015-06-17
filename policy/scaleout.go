package policy

import (
	"github.com/elleFlorio/gru/autonomic/analyzer"
	"github.com/elleFlorio/gru/service"
)

type ScaleOut struct {
}

func (p *ScaleOut) Name() string {
	return "scaleout"
}

func (p *ScaleOut) Type() string {
	return "proactive"
}

func (p *ScaleOut) Actions() []string {
	return []string{
		"start",
	}
}

func (p *ScaleOut) Weight(s *service.Service, a *analyzer.GruAnalytics) float64 {
	weight := 0.0
	cpuAvg := a.Service[s.Name].CpuAvg
	curActive := len(a.Service[s.Name].Instances.Running)
	maxActive := s.Constraints.MaxActive

	if curActive < maxActive {
		// Just for test...
		//weight = rand.Float64()
		if cpuAvg > s.Constraints.CpuMax {
			weight = (cpuAvg - s.Constraints.CpuMax) / (1.0 - s.Constraints.CpuMax)
		}
	}

	return weight
}

func (p *ScaleOut) Target() string {
	return "container"
}

func (p *ScaleOut) TargetStatus() string {
	return "stopped"
}