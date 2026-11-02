package tests

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/lynxland/lynx"
	"github.com/lynxland/lynx-clean-template/internal/pkg/config"
	"github.com/lynxland/lynx/contrib/zap"
)

type TestingSuite struct {
	App lynx.App
}

func NewTestingSuite() *TestingSuite {
	return &TestingSuite{}
}

type TestOptions struct {
	PreWaitTime  time.Duration
	PostWaitTime time.Duration
	LogToFile    bool
}

type TestOption func(*TestOptions)

// WithPostWaitTime 执行结束后等待的时间，等待其他 Component 关闭
func WithPostWaitTime(waitTime time.Duration) TestOption {
	return func(o *TestOptions) {
		o.PostWaitTime = waitTime
	}
}

func WithLogToFile() TestOption {
	return func(o *TestOptions) {
		o.LogToFile = true
	}
}

func WithPreWaitTime(waitTime time.Duration) TestOption {
	return func(o *TestOptions) {
		o.PreWaitTime = waitTime
	}
}

func newFileLogger(ctx lynx.AppContext) (*slog.Logger, error) {
	logLevel := ctx.Config().GetString("test.log-level")
	if logLevel == "" {
		logLevel = "info"
	}
	logFile := ctx.Config().GetString("test.log-file")
	if logFile == "" {
		logFile = "test.log"
	}
	zlogger, err := zap.NewZapLogger(logLevel, logFile)
	if err != nil {
		return nil, err
	}
	slogger, err := zap.NewSLogger(zlogger, logLevel)
	if err != nil {
		return nil, err
	}
	return slogger, nil
}

// RunTestSuite 初始化并运行测试套件
func RunTestSuite(fn func(ctx context.Context, ts *TestingSuite) error, opts ...TestOption) {
	buildTestSuite(fn, opts...).Run()
}

func buildTestSuite(fn func(ctx context.Context, ts *TestingSuite) error, opts ...TestOption) *lynx.Runner {
	// 测试选项在构造期生效（logger 经 WithLoggerProvider 注入，见下）。
	o := &TestOptions{
		PreWaitTime:  10 * time.Millisecond,
		PostWaitTime: 10 * time.Millisecond,
		LogToFile:    false,
	}
	for _, opt := range opts {
		opt(o)
	}
	return lynx.NewRunner(func(app lynx.App) error {
		ts, cleanup, err := wireTestingSuite(app)
		if err != nil {
			return err
		}
		ts.App = app

		// Wire cleanup 在所有服务停止后释放 DI 底层资源（OnPostStop）。
		app.OnPostStop(cleanup)

		app.Command(func(ctx context.Context) error {
			if o.PreWaitTime > 0 {
				slog.InfoContext(ctx, fmt.Sprintf("waiting %s for components startup", o.PreWaitTime.String()))
				time.Sleep(o.PreWaitTime)
			}

			if err := fn(ctx, ts); err != nil {
				slog.ErrorContext(ctx, "test execution error", "error", err)
				return fmt.Errorf("test execution error: %v", err)
			}

			if o.PostWaitTime > 0 {
				slog.InfoContext(ctx, fmt.Sprintf("waiting %s for components shutdown", o.PostWaitTime.String()))
				time.Sleep(o.PostWaitTime)
			}

			return nil
		})
		return nil
	},
		// logger 经构造期 provider 注入（v1.16.0 起 SetLogger 移除）：LogToFile
		// 走文件 logger（test.log-level / test.log-file 配置项），否则 contrib/zap。
		append(newTestOptions(), lynx.WithLoggerProvider(func(ctx lynx.AppContext) (*slog.Logger, error) {
			if o.LogToFile {
				return newFileLogger(ctx)
			}
			return zap.MustNewLogger(ctx), nil
		}))...)
}

func newTestOptions() []lynx.Option {
	return []lynx.Option{
		lynx.WithName("lynx:test"),
		lynx.WithBindConfigFunc(config.NewBindConfigFunc("./configs", "../configs", "../../configs")),
	}
}
