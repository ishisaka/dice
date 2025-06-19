package main

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutlog"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/trace"
)

// setupOTelSDK は OpenTelemetry SDK を初期化し、クリーンアップ関数を返します。
// ctx は初期化時のコンテキストを提供します。
// クリーンアップ関数を呼び出してリソースを解放することができます。
// 初期化中にエラーが発生した場合、そのエラーも返されます。
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
	tracerProvider, err := newTracerProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	// トレースプロバイダの登録
	otel.SetTracerProvider(tracerProvider)

	// 計装プロバイダーの作成
	meterProvider, err := newMeterProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, meterProvider.Shutdown)
	// 計装プロバイダーの登録
	otel.SetMeterProvider(meterProvider)

	// ログプロバイダーの作成
	loggerProvider, err := newLoggerProvider()
	if err != nil {
		handleErr(err)
		return
	}
	shutdownFuncs = append(shutdownFuncs, loggerProvider.Shutdown)
	// ログプロバイダーの登録
	global.SetLoggerProvider(loggerProvider)

	return
}

// newPropagator は OpenTelemetry の TextMapPropagator を作成して返します。
// TraceContext と Baggage を組み合わせたコンポジットプロパゲータを使用します。
func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

// newTracerProvider は新しい OpenTelemetry TracerProvider を作成して返します。
// 標準出力エクスポーターとバッチ処理を使用します。
// 初期化に失敗した場合はエラーを返します。
func newTracerProvider() (*trace.TracerProvider, error) {
	traceExporter, err := stdouttrace.New(
		stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter,
			// Default is 5s. Set to 1s for demonstrative purposes.
			trace.WithBatchTimeout(time.Second)),
	)
	return tracerProvider, nil
}

// newMeterProvider は、新しい OpenTelemetry 計装プロバイダーを作成して返します。
// 標準出力エクスポーターを使用し、データ収集間隔は3秒に設定されます。
// 初期化に失敗した場合はエラーを返します。
func newMeterProvider() (*metric.MeterProvider, error) {
	// 標準出力への出力を設定
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return nil, err
	}

	// 計装プロバイダーの作成
	meterProvider := metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			// デフォルトは1秒です。デモ目的で3秒に設定しています。
			metric.WithInterval(3*time.Second))),
	)
	return meterProvider, nil
}

// newLoggerProvider は新しい OpenTelemetry ログプロバイダーを作成して返します。
// 標準出力エクスポーターとバッチ処理を使用します。
// 初期化に失敗した場合はエラーを返します。
func newLoggerProvider() (*log.LoggerProvider, error) {
	// 標準出力への出力を設定
	logExporter, err := stdoutlog.New()
	if err != nil {
		return nil, err
	}

	// ログプロバイダーの作成
	loggerProvider := log.NewLoggerProvider(
		log.WithProcessor(log.NewBatchProcessor(logExporter)),
	)
	return loggerProvider, nil
}
