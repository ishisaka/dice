# dice
otelの練習

基本的に以下のチュートリアルをやるだけ。

[Getting Started \| OpenTelemetry](https://opentelemetry.io/docs/languages/go/getting-started/)

## ディレクトリ構造

- cmd
  - dice: otelのサンプルプログラム。現状はautoexportを使用したサンプルになっている
  - metrics: metricsのサンプル

## 参考(メモ)

- [autoexport package \- go\.opentelemetry\.io/contrib/exporters/autoexport \- Go Packages](https://pkg.go.dev/go.opentelemetry.io/contrib/exporters/autoexport@v0.61.0#NewSpanExporter)
- [otlpmetricgrpc package \- go\.opentelemetry\.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc \- Go Packages](https://pkg.go.dev/go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc)
- [What is OpenTelemetry? \| OpenTelemetry](https://opentelemetry.io/docs/what-is-opentelemetry/)
- [Standalone \.NET Aspire dashboard \- \.NET Aspire \| Microsoft Learn](https://learn.microsoft.com/en-us/dotnet/aspire/fundamentals/dashboard/standalone?tabs=bash)