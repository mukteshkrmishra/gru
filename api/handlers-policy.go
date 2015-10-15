package api

import (
	"encoding/json"
	"net/http"

	log "github.com/elleFlorio/gru/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/elleFlorio/gru/autonomic/planner/policy"
)

type plc struct {
	Name    string   `json:"name"`
	Actions []string `json:"actions"`
}

// /gru/v1/policies
func GetInfoPolicies(w http.ResponseWriter, r *http.Request) {
	policies := policy.GetPolicies()
	plcs := createPoliciesJson(policies)

	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(plcs); err != nil {
		log.WithFields(log.Fields{
			"status":  "http response",
			"request": "GetInfoPolicies",
			"error":   err,
		}).Errorln("API Server")
	}
}

func createPoliciesJson(policies []policy.GruPolicy) []plc {
	plcs := make([]plc, 0, len(policies))

	for _, p := range policies {
		plc_actions := []string{}
		for _, action := range p.Actions() {
			plc_actions = append(plc_actions, action.ToString())
		}
		plc_tmp := plc{
			p.Name(),
			plc_actions,
		}
		plcs = append(plcs, plc_tmp)
	}

	return plcs
}
