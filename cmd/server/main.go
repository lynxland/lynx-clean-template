package main

import (
	"log/slog"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/lynxland/lynx"
	config "github.com/lynxland/lynx-clean-template/internal/pkg/config"
	"github.com/lynxland/lynx-clean-template/pkg/timeutil"
	"github.com/lynxland/lynx/contrib/zap"
	"github.com/spf13/pflag"
)

var (
	version string
)

func main() {
	runner := lynx.NewRunner(func(app lynx.App) error {
		boot, cleanup, err := wireBootstrap(app)
		if err != nil {
			return err
		}
		// Wire cleanup 释放 DI 底层资源（DB/Redis 连接池等），必须在所有
		// 服务停止之后执行——挂 OnPostStop（v1.10.0 前挂 OnStop 会在在途
		// 请求收尾前关掉连接池）。
		app.OnPostStop(cleanup)
		boot.Apply(app)
		return nil
	},
		// logger 经构造期 provider 注入（v1.16.0 起 SetLogger 移除）：总线及
		// 配套服务在装配序列内捕获同一 logger，装配期与运行期格式一致。
		lynx.WithLoggerProvider(func(ctx lynx.AppContext) (*slog.Logger, error) {
			return zap.MustNewLogger(ctx), nil
		}),
		lynx.WithName("lynx-api"),
		lynx.WithVersion(version),
		lynx.WithBindFlagsFunc(func(f *pflag.FlagSet) {
			f.String("config-dir", "./configs", "config file path")
			f.String("log-level", "info", "log level, default info")
		}),
		lynx.WithBindConfigFunc(config.NewBindConfigFunc()),
		lynx.WithShutdownTimeout(30*time.Second),
	)
	runner.Run()
}

func init() {
	// Load .env file if present (dev convenience; silently ignored in production).
	_ = godotenv.Load()

	timeutil.InitCarbon()
}
