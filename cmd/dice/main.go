package main

import (
	"context"
	"errors"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"log"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/samber/slog-multi"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var logger *slog.Logger

func main() {
	logLevl := slog.LevelDebug
	logger = slog.New(
		slogmulti.Fanout(
			otelslog.NewHandler("dice"),
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: logLevl}),
		),
	)
	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

// run はSIGINTシグナルをハンドルしながらHTTPサーバーとOpenTelemetryの初期化・クリーンアップを管理します。
func run() (err error) {
	// SIGINT（CTRL+C）のハンドル
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	// オープンテレメトリの設定
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}
	// シャットダウンを適切に処理し、データがリークしないようにする
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Start HTTP server.
	srv := &http.Server{
		Addr:         ":8080",
		BaseContext:  func(_ net.Listener) context.Context { return ctx },
		ReadTimeout:  time.Second,
		WriteTimeout: 10 * time.Second,
		Handler:      newHTTPHandler(),
	}
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- srv.ListenAndServe()
	}()
	logger.Info("HTTP server is listening on :8080")

	// 中断処理
	select {
	case err = <-srvErr:
		// HTTP serverでエラーが発生したとき
		return
	case <-ctx.Done():
		// 最初のCTRL+Cを待つ
		// できるだけ早く信号通知の受信を停止する
		stop()
	}

	// シャットダウンが呼び出されると、ListenAndServeは即座にErrServerClosedを返す
	err = srv.Shutdown(context.Background())
	return
}

// newHTTPHandler はHTTPリクエストを処理する新しいhttp.Handlerを作成して返します。
// 登録されたエンドポイントとOpenTelemetryのHTTP計装を設定します。
func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	// handleFunc は、mux.HandleFunc の代替機能で、ハンドラーの HTTP ログ記録機能を
	// http.route のパターンに基づいて拡張します。
	handleFunc := func(pattern string, handlerFunc func(http.ResponseWriter, *http.Request)) {
		// Configure the "http.route" for the HTTP instrumentation.
		handler := otelhttp.WithRouteTag(pattern, http.HandlerFunc(handlerFunc))
		mux.Handle(pattern, handler)
	}

	// ハンドラーの関数を設定
	handleFunc("/rolldice/", rolldice)
	handleFunc("/rolldice/{player}", rolldice)

	// サーバー全体にHTTP監視機能を追加します。
	handler := otelhttp.NewHandler(mux, "/")
	return handler
}
