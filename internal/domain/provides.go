package domain

import (
	"github.com/google/wire"
	"github.com/lynxland/lynx-clean-template/internal/domain/files"
	"github.com/lynxland/lynx-clean-template/internal/domain/users"
)

var ProviderSet = wire.NewSet(
	users.NewUserService,
	files.NewService,
	files.NewURLRenderer,
)
