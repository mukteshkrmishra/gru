package policy

import (
	"math"

	cfg "github.com/elleFlorio/gru/configuration"
	"github.com/elleFlorio/gru/data"
	"github.com/elleFlorio/gru/enum"
	res "github.com/elleFlorio/gru/resources"
	srv "github.com/elleFlorio/gru/service"
	"github.com/elleFlorio/gru/utils"
)

type scaleoutCreator struct{}

func (p *scaleoutCreator) getPolicyName() string {
	return "scaleout"
}

func (p *scaleoutCreator) listActions() []string {
	return []string{"start"}
}

func (p *scaleoutCreator) createPolicies(srvList []string, clusterData data.Shared) []data.Policy {
	scaleoutPolicies := make([]data.Policy, 0, len(srvList))
	if !cfg.GetPolicy().Scaleout.Enable {
		return scaleoutPolicies
	}

	for _, name := range srvList {
		policyName := p.getPolicyName()
		policyWeight := p.computeWeight(name, clusterData)
		policyTargets := []string{name}
		policyActions := map[string][]enum.Action{
			name: []enum.Action{enum.START},
		}

		scaleoutPolicy := data.Policy{
			Name:    policyName,
			Weight:  policyWeight,
			Targets: policyTargets,
			Actions: policyActions,
		}

		scaleoutPolicies = append(scaleoutPolicies, scaleoutPolicy)
	}

	return scaleoutPolicies
}

func (p *scaleoutCreator) computeWeight(name string, clusterData data.Shared) float64 {
	service, _ := srv.GetServiceByName(name)

	if res.AvailableResourcesService(name) < 1.0 {
		return 0.0
	}

	srvCores := service.Docker.CpusetCpus
	if srvCores != "" {
		if !res.CheckSpecificCoresAvailable(srvCores) {
			return 0.0
		}
	}

	analytics := srv.GetServiceExpressionsList(name)
	threshold := cfg.GetPolicy().Scaleout.Threshold
	weights := []float64{}

	for _, value := range clusterData.Service[name].Data.BaseShared {
		weights = append(weights, p.computeMetricWeight(value, threshold))
	}

	for _, analytic := range analytics {
		value := clusterData.Service[name].Data.UserShared[analytic]
		weights = append(weights, p.computeMetricWeight(value, threshold))
	}

	policyValue := utils.Mean(weights)

	return policyValue
}

func (p *scaleoutCreator) computeMetricWeight(value float64, threshold float64) float64 {
	return 1 - (math.Max(value, threshold) / (1 - threshold))
}
