package server

import (
	"path/filepath"
	"testing"

	"github.com/lynxland/grpcapi/guard"
	"github.com/lynxland/grpcapi/policy/protopolicy"
)

// swaggerErrorResponseRef 与各服务 proto 文件级 openapiv2_swagger.responses.default
// 声明的引用经生成器解析后的定义名保持一致（legacy 命名：lynx.shared 包经
// 最短唯一后缀落在 shared 前缀下）。
const swaggerErrorResponseRef = "#/definitions/sharedErrorResponse"

// serviceDefaultLabel 从 proto service 策略注解解析服务默认文档 label（与启动期
// policy.Build 的回落语义同源）——protopolicy.Service 的 Label 即规范形态，
// 无需项目自写扩展解析与字符串映射。
func serviceDefaultLabel(service string) (string, bool) {
	prov := protopolicy.New(protopolicy.Options{})
	for _, fd := range authzFileDescriptors() {
		for i := 0; i < fd.Services().Len(); i++ {
			s := fd.Services().Get(i)
			if string(s.FullName()) == service {
				spec, ok, err := prov.Service(s)
				if err != nil || !ok {
					return "", false
				}
				return spec.Label, true
			}
		}
	}
	return "", false
}

// TestSwaggerAccessMatchesPolicies 断言 genproto/api/v1/*.swagger.json 的
// x-access 扩展与 policy.Set 完全一致（README 第 6 步守卫）：顶层扩展
// == 服务默认 label；operation 有效扩展（显式否则继承顶层）== 方法有效
// label；default 响应引用运行时真实错误体；策略表方法反向全覆盖。前提：
// 仓库根执行过 mise run generate:proto（genproto 已入库）。
func TestSwaggerAccessMatchesPolicies(t *testing.T) {
	policies, err := NewPolicySet()
	if err != nil {
		t.Fatalf("NewPolicySet: %v", err)
	}

	guard.SwaggerAccessMatches(t, guard.SwaggerOptions{
		GenprotoRoot:     filepath.Join("..", "..", "..", "genproto"),
		GenprotoSubdirs:  []string{"api/v1"},
		Files:            authzFileDescriptors(),
		Policies:         policies,
		AccessExtension:  "x-access",
		ErrorResponseRef: swaggerErrorResponseRef,
		MinFiles:         4,
		MinOps:           14,
		ServiceAccess:    serviceDefaultLabel,
	})

	// RPC 快照锁：lynx.api.v1 下方法总数（AuthService 4 + UsersService 4 +
	// GroupsService 5 + FilesService 1 = 14）。增删 RPC 请同步更新此常量。
	guard.AssertMethodCount(t, "lynx.api.v1", 14)
}

// TestSwaggerPropertyNamesAreSnakeCase 守卫 buf.gen.yaml 的
// json_names_for_fields=false（与运行时 marshaler UseProtoNames 对齐）。
func TestSwaggerPropertyNamesAreSnakeCase(t *testing.T) {
	guard.SwaggerSnakeCaseProperties(t,
		filepath.Join("..", "..", "..", "genproto"), []string{"api/v1"})
}

// TestSwaggerNoRpcStatus 守卫 buf.gen.yaml 的 disable_default_errors=true
// （生成器默认注入的 rpcStatus 与运行时错误体 lynx.shared.ErrorResponse 不符，
// 配置回退即红）。
func TestSwaggerNoRpcStatus(t *testing.T) {
	guard.SwaggerNoDefinition(t,
		filepath.Join("..", "..", "..", "genproto"), []string{"api/v1"}, "rpcStatus")
}

// TestBuildMatchesContract 等价锁：注解收集结果与鉴权意图一致——
// anonymous（public 档，拦截器放行）vs 认证要求（拦截器做端用户认证；
// permission 档的角色判定在 app 用例层），label 与 swagger x-access 契约
// 逐字对齐（public / end_user / permission）。
func TestBuildMatchesContract(t *testing.T) {
	set, err := NewPolicySet()
	if err != nil {
		t.Fatalf("NewPolicySet: %v", err)
	}

	want := map[string]string{
		"/lynx.api.v1.AuthService/Token":                 "public",
		"/lynx.api.v1.AuthService/SignUp":                "public",
		"/lynx.api.v1.AuthService/VerifySignUpEmail":     "public",
		"/lynx.api.v1.AuthService/ResendSignUpEmailCode": "public",
		"/lynx.api.v1.UsersService/UpdateMyProfile":      "end_user",
		"/lynx.api.v1.UsersService/GetUserProfile":       "end_user",
		"/lynx.api.v1.UsersService/GrantSuperAdmin":      "permission",
		"/lynx.api.v1.UsersService/RevokeSuperAdmin":     "permission",
		"/lynx.api.v1.GroupsService/CreateGroup":         "end_user",
		"/lynx.api.v1.GroupsService/GetGroup":            "end_user",
		"/lynx.api.v1.GroupsService/ListGroups":          "end_user",
		"/lynx.api.v1.GroupsService/UpdateGroup":         "end_user",
		"/lynx.api.v1.GroupsService/DeleteGroup":         "end_user",
		"/lynx.api.v1.FilesService/GetPresignedURL":      "end_user",
	}
	for method, wantLabel := range want {
		op, ok := set.Get(method)
		if !ok {
			t.Fatalf("missing policy for %s", method)
		}
		if got := op.EffectiveLabel(); got != wantLabel {
			t.Errorf("%s: label = %q, want %q", method, got, wantLabel)
		}
		// public 档 == anonymous 策略（拦截器据此放行，行为与 v0.1
		// AccessPublic 一致）；其余档位必须要求认证。
		if anonymous := op.Policy.Anonymous; anonymous != (wantLabel == "public") {
			t.Errorf("%s: anonymous = %v, want %v（label %q）", method, anonymous, wantLabel == "public", wantLabel)
		}
	}

	// 模板无 API key 面：任何方法不得携带 scope 要求；全平面为一元 RPC。
	for _, op := range set.Operations() {
		for _, req := range op.Policy.Require {
			if req.Kind == "scope" {
				t.Errorf("%s: scope 要求 %v 不期而至（模板无 API key 面）", op.ID, req.Values)
			}
		}
		if op.Streaming {
			t.Errorf("%s: 不应为流式方法", op.ID)
		}
	}
}
