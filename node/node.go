package node

import (
	"encoding/json"
	"io/ioutil"

	log "github.com/elleFlorio/gru/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/elleFlorio/gru/container"
	"github.com/elleFlorio/gru/utils"
)

var config Node

func LoadNodeConfig(filename string) error {
	config.UUID, _ = utils.GenerateUUID()

	log.WithField("status", "start").Infoln("Node configuration loading")
	defer log.WithFields(log.Fields{
		"status": "done",
		"UUID":   config.UUID,
	}).Infoln("Node configuration loading")

	tmp, err := ioutil.ReadFile(filename)
	if err != nil {
		log.WithField("error", err).Errorln("Error reading node configuration file")
		return err
	}
	err = json.Unmarshal(tmp, &config)
	if err != nil {
		log.WithField("error", err).Errorln("Error unmarshaling node configuration file")
		return err
	}

	computeTotalResources()

	return nil
}

func computeTotalResources() {
	info, err := container.Docker().Client.Info()
	if err != nil {
		log.WithField("error", err).Errorln("Error reading total resources")
		return
	}
	config.Resources.TotalCpus = info.NCPU
	config.Resources.TotalMemory = info.MemTotal
}

func UsedCpus() (int64, error) {
	var cpus int64

	containers, err := container.Docker().Client.ListContainers(false, false, "")
	if err != nil {
		return 0, err
	}

	for _, c := range containers {
		cData, err := container.Docker().Client.InspectContainer(c.Id)
		if err != nil {
			return 0, err
		}

		cpus += cData.Config.CpuShares
	}

	config.Resources.UsedCpu = cpus

	return cpus, nil
}

func UsedMemory() (int64, error) {
	var memory int64

	containers, err := container.Docker().Client.ListContainers(false, false, "")
	if err != nil {
		return 0, err
	}

	for _, c := range containers {
		cData, err := container.Docker().Client.InspectContainer(c.Id)
		if err != nil {
			return 0, err
		}

		memory += cData.Config.Memory
	}

	config.Resources.UsedMemory = memory

	return memory, nil
}

func Config() Node {
	return config
}

func UpdateNodeConfig(newNode Node) {
	config = newNode
}
