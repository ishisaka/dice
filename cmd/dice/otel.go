package main

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/contrib/exporters/autoexport"
	"go.opentelemetry.io/contrib/processors/minsev"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.32.0"
)

// setupOTelSDK は OpenTelemetry SDK を初期化し、シャットダウン用のクリーンアップ関数を返します。
// 渡されたコンテキストを使用してプロバイダーを設定し、エラー発生時は適切なクリーンアップを行います。
// 初期化には、トレース、計装、ログの各プロバイダーが含まれます。また、関連するクリーンアップ関数を登録します。
// シャットダウン関数は一度だけ実行され、実行中に発生した全てのエラーを結合して返します。
func setupOTelSDK(ctx context.Context) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown 関数は、shutdownFuncs経由で登録されたクリーンアップ関数を実行します。
	// 呼び出しから発生したエラーは結合されます。
	// 各登録されたクリーンアップ関数は1回だけ実行されます。
	shutdown = func(ctx context.Context) error {
		var err error
		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}
		shutdownFuncs = nil
		return err
	}

	// handleErr はシャットダウンを実行してクリーンアップを行い、
	// すべてのエラーが返されることを確認します。
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	// OpenTelemetry プロパゲーターは、アプリケーション間で交換されるメッセージからコンテキストデ
	// ータを抽出したり、メッセージに注入したりするために使用されます。
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	// トレースプロバイダの作成
	tracerProvider, err := newTracerProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	// トレースプロバイダの登録
	otel.SetTracerProvider(tracerProvider)

	// 計装プロバイダーの作成
	meterProvider, err := newMeterProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	// 計装プロバイダーの登録
	otel.SetMeterProvider(meterProvider)

	// ログプロバイダーの作成
	loggerProvider, err := newLoggerProvider(ctx)
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	// ログプロバイダーの登録
	global.SetLoggerProvider(loggerProvider)

	return
}

// newPropagator は新しい TextMapPropagator を作成して返します。Traces と Baggage のプロパゲーションをサポートします。
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTracerProvider は新しい OpenTelemetry TracerProvider を作成して返します。
// 標準出力エクスポーターとバッチ処理を使用します。
// 初期化に失敗した場合はエラーを返します。
func newTracerProvider(ctx context.Context) (*trace.TracerProvider, error) {
	traceExporter, err := autoexport.NewSpanExporter(ctx)
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithResource(getResource()),
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return tracerProvider, nil
}

// newMeterProvider は、新しい OpenTelemetry 計装プロバイダーを作成して返します。
// 標準出力エクスポーターを使用し、データ収集間隔は3秒に設定されます。
// 初期化に失敗した場合はエラーを返します。
func newMeterProvider(ctx context.Context) (*metric.MeterProvider, error) {
	// 標準出力への出力を設定
	metricReader, err := autoexport.NewMetricReader(ctx)
	if err != nil {
		return nil, err
	}

	// 計装プロバイダーの作成
	meterProvider := metric.NewMeterProvider(
		metric.WithResource(getResource()),
		metric.WithReader(metricReader),
	)
	return meterProvider, nil
}

// newLoggerProvider は新しい OpenTelemetry ログプロバイダーを作成して返します。
// 標準出力エクスポーターとバッチ処理を使用します。
// 初期化に失敗した場合はエラーを返します。
func newLoggerProvider(ctx context.Context) (*log.LoggerProvider, error) {
	// 標準出力への出力を設定
	logExporter, err := autoexport.NewLogExporter(ctx)
	if err != nil {
		return nil, err
	}

	// ログレベルをminsevで指定
	// go.opentelemetry.io/contrib/processors/minsev
	loglevel := minsev.SeverityDebug

	// ログプロバイダーの作成
	loggerProvider := log.NewLoggerProvider(
		log.WithResource(getResource()),
		log.WithProcessor(
			minsev.NewLogProcessor(
				log.NewBatchProcessor(logExporter),
				loglevel,
			),
		),
	)
	return loggerProvider, nil
}

// getResource はリソース情報を生成し、サービス名、バージョン、インスタンスIDを含むリソースを返します。
func getResource() *resource.Resource {
	res := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String("dice"),
		semconv.ServiceVersionKey.String("1.0.0"),
		semconv.ServiceInstanceIDKey.String("abcdef12345"),
	)
	return res
}
