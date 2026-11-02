package interceptor

import (
	"context"
	"errors"
	"fmt"
	"strings"

	grpcapiinterceptor "github.com/lynxland/grpcapi/interceptor"
	"github.com/lynxland/grpcapi/policy"
	"github.com/lynxland/lynx-clean-template/internal/pkg/contexts"
	"github.com/lynxland/lynx-clean-template/pkg/jwtparser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type (
	// Validator defines an interface for token validation. This is satisfied by our auth service.
	Validator interface {
		ValidateToken(ctx context.Context, token string) (string, error)
	}

	// AuthInterceptor 是 grpcapi 拦截器链的 authn Stage 实现：不再维护硬编码
	// skip 列表，改为按注入的 policy.Set（proto 注解的运行时投影）逐方法
	// 判定——proto 是鉴权策略唯一声明源。
	AuthInterceptor struct {
		validator Validator
		policies  *policy.Set
	}
)

// NewAuthInterceptor 构造 auth 拦截器。policies 由 server.NewPolicySet
// 从 proto 注解收集（构造期已 fail-closed），此处按方法策略分流：
//   - anonymous：跳过认证（原 PUBLIC 档语义，声明点上移到 proto）；
//   - 其余（authenticated/角色等要求）：JWT 校验并注入用户身份。声明了
//     roles/permissions 的方法（如 GrantSuperAdmin 的 super_admin）拦截器
//     只做端用户认证；角色权限判定交由 app 用例层把关（users.IsSuperAdmin
//     已有该检查），避免拦截器与 app 层两套权限语义漂移。
func NewAuthInterceptor(validator Validator, policies *policy.Set) (*AuthInterceptor, error) {
	if validator == nil {
		return nil, errors.New("validator cannot be nil")
	}
	if policies == nil {
		return nil, errors.New("policies cannot be nil")
	}
	return &AuthInterceptor{validator: validator, policies: policies}, nil
}

func (i *AuthInterceptor) UnaryAuthMiddleware(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// 框架内置服务（grpc.health.v1 / grpc.reflection.*）不携带业务注解，
	// 由部署层网络策略保护，与库内 validate 拦截器共用同一豁免表。
	if grpcapiinterceptor.FrameworkExempt(info.FullMethod) {
		return handler(ctx, req)
	}

	if !i.policies.IsAnonymous(info.FullMethod) {
		if _, ok := i.policies.Get(info.FullMethod); !ok {
			// fail-closed 双保险：启动断言 AssertAllRegisteredHavePolicy 应已
			// 拦截漏配，运行期再兜底一次（漏配方法拒绝而非放行）。
			return nil, status.Errorf(codes.PermissionDenied, "missing auth policy for %s", info.FullMethod)
		}
		return i.authenticate(ctx, req, info, handler)
	}
	return handler(ctx, req)
}

// authenticate 校验 Bearer JWT 并把用户身份注入 ctx，随后放行 handler。
func (i *AuthInterceptor) authenticate(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	// get metadata object
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "metadata is not provided")
	}

	// extract token from authorization header
	tokens := md["authorization"]
	if len(tokens) == 0 {
		return nil, status.Error(codes.Unauthenticated, "authorization token is not provided")
	}

	token := tokens[0]
	token = strings.TrimPrefix(token, "Bearer ")
	// validate token and retrieve the userID
	userID, err := i.validator.ValidateToken(ctx, token)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, fmt.Sprintf("invalid token: %v", err))
	}

	// Inject canonical key consumed by app/contexts helpers.
	ctx = context.WithValue(ctx, contexts.ContextKeyCurrentUser, userID)
	// Keep legacy key during migration to avoid breaking older call paths.
	ctx = context.WithValue(ctx, "_USER", userID)

	// call our handler
	return handler(ctx, req)
}

type DefaultValidator struct {
	secretKey string
}

func (v *DefaultValidator) ValidateToken(ctx context.Context, token string) (string, error) {
	userId, _, _, _, _, _, ok := jwtparser.ParseToken([]byte(v.secretKey), token)
	if !ok {
		return "", status.Error(codes.Unauthenticated, "invalid token")
	}
	return userId, nil
}

func NewDefaultValidator(
	secretKey string,
) Validator {
	return &DefaultValidator{secretKey: secretKey}
}
