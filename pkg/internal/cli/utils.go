package cli

import (
	"encoding/json"

	"code.google.com/p/go-uuid/uuid"
	log "github.com/Sirupsen/logrus"
)

// AssertID checks whether a string is a valid id
func AssertID(id string) {
	if uuid := uuid.Parse(id); uuid == nil {
		log.WithFields(log.Fields{
			"id": id,
		}).Fatal("invalid id")
	}
}

// AssertSpec checks whether a json string parses as expected
func AssertSpec(spec string) {
	j := JMap{}
	if err := json.Unmarshal([]byte(spec), &j); err != nil {
		log.WithFields(log.Fields{
			"spec":  spec,
			"error": err,
		}).Fatal("invalid spec")
	}
}
