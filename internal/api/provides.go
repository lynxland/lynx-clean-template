package api

import (
	"github.com/google/wire"
	"github.com/lynxland/lynx-clean-template/internal/api/cronjob"
	"github.com/lynxland/lynx-clean-template/internal/api/eventhandler"
	"github.com/lynxland/lynx-clean-template/internal/api/grpc"
)

var ProviderSet = wire.NewSet(
	cronjob.NewDemoTask,
	eventhandler.NewHelloHandler,
	grpc.NewAuthService,
	grpc.NewUsersService,
	grpc.NewGroupsService,
)
