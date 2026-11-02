package events

import (
	"time"

	"github.com/lynxland/lynx-clean-template/pkg/idgen"
)

type HelloEvent struct {
	User    idgen.ID  `json:"user"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}
