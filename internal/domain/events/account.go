package events

import (
	"time"

	"github.com/lynxland/lynx-clean-template/pkg/idgen"
)

type AccountCreatedEvent struct {
	UserID      idgen.ID  `json:"user_id"`
	Username    string    `json:"username"`
	DisplayName string    `json:"display_name"`
	Time        time.Time `json:"time"`
}

type AccountAuthenticatedEvent struct {
	UserID idgen.ID  `json:"user_id"`
	Time   time.Time `json:"time"`
}
