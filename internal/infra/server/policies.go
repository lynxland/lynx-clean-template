package server

import (
	"context"

	"github.com/lynxland/grpcapi/policy"
	"github.com/lynxland/grpcapi/policy/protopolicy"
	apipb "github.com/lynxland/lynx-clean-template/genproto/api/v1"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// authzFileDescriptors 是业务 proto 文件描述符的单一事实源：启动期策略收集
// （NewPolicySet）与守卫测试（swagger access 对齐检查）共用同一清单。
// FilesService 当前未注册到 gRPC server，但策略先行收集——注册后即被
// AssertAllRegisteredHavePolicy 覆盖，无需再补注解。
func authzFileDescriptors() []protoreflect.FileDescriptor {
	return []protoreflect.FileDescriptor{
		apipb.File_api_v1_auth_proto,
		apipb.File_api_v1_users_proto,
		apipb.File_api_v1_groups_proto,
		apipb.File_api_v1_files_proto,
	}
}

// NewPolicySet 从 proto 注解（grpcapi.v1 的 api/service/method 策略扩展，
// 经 protopolicy provider 读取）收集全量 API 授权策略，构造期 fail-closed
// （缺注解、策略校验失败等启动即失败）。模板没有 scope 词表面，
// Vocabulary 缺省为空是合法路径（空词表断言仅在存在 scope 声明时触发）。
func NewPolicySet() (*policy.Set, error) {
	return policy.Build(context.Background(),
		policy.FromFiles(authzFileDescriptors()...),
		policy.Options{
			Provider: protopolicy.New(protopolicy.Options{}),
		})
}
