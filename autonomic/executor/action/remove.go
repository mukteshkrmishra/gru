package action

import (
	"errors"

	log "github.com/elleFlorio/gru/Godeps/_workspace/src/github.com/Sirupsen/logrus"

	"github.com/elleFlorio/gru/container"
	"github.com/elleFlorio/gru/enum"
)

var errNoContainerToRemove = errors.New("No stopped container to remove")

type Remove struct{}

func (p *Remove) Type() enum.Action {
	return enum.REMOVE
}

func (p *Remove) Run(config Action) error {
	var err error
	stopped := config.Instances.Stopped
	if len(stopped) < 1 {
		log.WithFields(log.Fields{
			"service": config.Service,
			"err":     errNoContainerToRemove,
		}).Errorln("Cannot remove container")

		return errNoContainerToRemove
	}

	toRemove := stopped[0]
	// Assumption: I remove only stopped containers; containers have no volume
	err = container.Docker().Client.RemoveContainer(toRemove, false, false)
	if err != nil {
		log.WithFields(log.Fields{
			"service":  config.Service,
			"instance": toRemove,
			"err":      err,
		}).Errorln("Cannot remove container")

		return err
	}

	log.WithFields(log.Fields{
		"service":  config.Service,
		"instance": toRemove,
	}).Debugln("Removed container")

	return nil
}
