package strategy

import (
	"encoding/json"
	"math/rand"

	"github.com/elleFlorio/gru/enum"
	"github.com/elleFlorio/gru/service"
	"github.com/elleFlorio/gru/storage"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func CreateMockPlan(w float64, s service.Service, a []enum.Action) GruPlan {
	return GruPlan{w, &s, a}
}

func StoreMockPlan(w float64, s service.Service, a []enum.Action) {
	plan := CreateMockPlan(w, s, a)
	data, _ := convertPlanToData(plan)
	storage.StoreLocalData(data, enum.PLANS)
}

func convertPlanToData(plan GruPlan) ([]byte, error) {
	data, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func CreateRandomPlans(n int) []GruPlan {
	plans := []GruPlan{}
	for i := 0; i < n; i++ {
		value := randUniform(0, 1)
		w := value
		s := service.Service{Name: randStringBytes(5)}
		a := []enum.Action{enum.START}
		if value > 0.5 {
			a = []enum.Action{enum.STOP}
		}
		plans = append(plans, GruPlan{w, &s, a})
	}

	return plans
}

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
