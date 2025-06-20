package main

import (
	"context"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

func main() {
	ctx := context.Background()
	exp, err := otlpmetricgrpc.New(ctx, otlpmetricgrpc.WithInsecure()) // 送信先などはここで設定できるが今回はlocalhostなので不要
	if err != nil {
		panic(err)
	}

	meterProvider := metric.NewMeterProvider(metric.WithReader(metric.NewPeriodicReader(exp)))
	defer func() {
		if err := meterProvider.Shutdown(ctx); err != nil {
			panic(err)
		}
	}()
	otel.SetMeterProvider(meterProvider)
	meter := otel.Meter("go.opentelemetry.io/otel/metric#MultiAsyncExample")

	// 計測する項目と概要を定義する
	cpuUsage, _ := meter.Int64ObservableGauge(
		"cpuUsage",
		api.WithDescription("CPU Usage in %"),
	)

	// コレクション中に呼び出される関数を登録する
	_, err = meter.RegisterCallback(
		func(_ context.Context, o api.Observer) error {
			memStats := &runtime.MemStats{}
			// This call does work
			runtime.ReadMemStats(memStats)
			o.ObserveInt64(cpuUsage,
				int64(60),
				api.WithAttributes(
					attribute.String("label", "value"), // 属性を設定
					attribute.Bool("env-prod", true),   // valueの型ごとにメソッドが用意されている
				),
			)
			return nil
		},
		cpuUsage,
	)
	if err != nil {
		fmt.Println("Failed to register callback")
		panic(err)
	}
}
