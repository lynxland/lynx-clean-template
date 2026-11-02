package server

import (
	grpcapiinterceptor "github.com/lynxland/grpcapi/interceptor"
	"github.com/lynxland/grpcapi/policy"
	"github.com/lynxland/lynx"
	apipb "github.com/lynxland/lynx-clean-template/genproto/api/v1"
	grpcsvc "github.com/lynxland/lynx-clean-template/internal/api/grpc"
	"github.com/lynxland/lynx-clean-template/internal/pkg/config"
	"github.com/lynxland/lynx-clean-template/pkg/grpc/interceptor"
	lynxgrpc "github.com/lynxland/lynx/server/grpc"
)

func NewAuthValidator(
	config *config.AppConfig,
) interceptor.Validator {
	return interceptor.NewDefaultValidator(config.Security.Jwt.Secret)
}

func NewGRPCServer(
	app lynx.App,
	cfg *config.AppConfig,
	authValidator interceptor.Validator,
	policies *policy.Set,
	auth *grpcsvc.AuthService,
	groups *grpcsvc.GroupsService,
	users *grpcsvc.UsersService,
) (*lynxgrpc.Server, error) {
	c := cfg.GetServer().GetGrpc()
	addr := c.Addr
	timeout := parseTimeout(c.Timeout)
	authInterceptor, err := interceptor.NewAuthInterceptor(authValidator, policies)
	if err != nil {
		return nil, err
	}

	// 拦截器链装配换库（grpcapi Stage 语义）：clientinfo 打头、validate 收尾
	//（DefaultChainOptions 构造期断言），authn 居中（Order 30）。模板原先没有
	// clientInfo 拦截器，经库免费获得（客户端 IP/UA 注入 ctx；模板 config 无
	// trusted_proxies 配置，默认不信任转发头，一律使用 gRPC peer 地址）。
	chain, err := grpcapiinterceptor.AssembleWith(grpcapiinterceptor.DefaultChainOptions(),
		grpcapiinterceptor.Stage{Name: grpcapiinterceptor.StageClientInfo, Order: grpcapiinterceptor.OrderClientInfo,
			Unary: grpcapiinterceptor.NewClientInfo(grpcapiinterceptor.ClientInfoConfig{}).Unary()},
		grpcapiinterceptor.Stage{Name: grpcapiinterceptor.StageAuthn, Order: grpcapiinterceptor.OrderAuthn,
			Unary: authInterceptor.UnaryAuthMiddleware},
		grpcapiinterceptor.Stage{Name: grpcapiinterceptor.StageValidate, Order: grpcapiinterceptor.OrderValidate,
			Unary: grpcapiinterceptor.NewValidate().Unary()},
	)
	if err != nil {
		return nil, err
	}

	srv := lynxgrpc.NewServer(lynxgrpc.WithAddr(addr), lynxgrpc.WithTimeout(timeout), lynxgrpc.WithLogger(app.Logger()), lynxgrpc.WithInterceptors(chain.Unary()...))
	grpcSrv := srv.GetServer()

	// Register public API services
	apipb.RegisterAuthServiceServer(grpcSrv, auth)
	apipb.RegisterGroupsServiceServer(grpcSrv, groups)
	apipb.RegisterUsersServiceServer(grpcSrv, users)

	// 启动断言（fail-closed）：每个已注册方法必须在策略集中，proto 注解与
	// 注册实现漂移启动即红。
	if err := grpcapiinterceptor.AssertAllRegisteredHavePolicy(grpcSrv, policies); err != nil {
		return nil, err
	}

	return srv, nil
}
