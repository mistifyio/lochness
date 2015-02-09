package cli

import (
	"encoding/json"

	log "github.com/Sirupsen/logrus"

	"code.google.com/p/go-uuid/uuid"
)

func AssertID(id string) {
	if uuid := uuid.Parse(id); uuid == nil {
		log.WithFields(log.Fields{
			"id": id,
		}).Fatal("invalid id")
	}
}

func AssertSpec(spec string) {
	j := JMap{}
	if err := json.Unmarshal([]byte(spec), &j); err != nil {
		log.WithFields(log.Fields{
			"spec":  spec,
			"error": err,
		}).Fatal("invalid spec")
	}
}
