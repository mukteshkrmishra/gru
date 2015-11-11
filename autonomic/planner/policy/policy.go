package policy

import (
	"github.com/elleFlorio/gru/autonomic/analyzer"
	"github.com/elleFlorio/gru/enum"
)

type GruPolicy interface {
	Name() string
	Label(string, analyzer.GruAnalytics) enum.Label
	Actions() []enum.Action
}

var policies []GruPolicy

func init() {
	policies = []GruPolicy{
		&ScaleIn{},
		&ScaleOut{},
	}
}

func GetPolicies() []GruPolicy {
	return policies
}

func List() []string {
	names := []string{}
	for _, policy := range policies {
		names = append(names, policy.Name())
	}

	return names
}