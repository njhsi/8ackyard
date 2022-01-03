/*

 */
package entity

import (
	"github.com/njhsi/8ackyard/internal/event"
)

var log = event.Log
var GeoApi = "places"

// Log logs the error if any and keeps quiet otherwise.
func Log(model, action string, err error) {
	if err != nil {
		log.Errorf("%s: %s (%s)", model, err, action)
	}
}
