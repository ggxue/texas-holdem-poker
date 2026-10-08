package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"texas-poker/internal/poker"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("缺少 DATABASE_URL：需要 PostgreSQL 存档连接")
	}
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return errors.New("DATABASE_URL 配置无效")
	}
	config.MaxConns = 4
	db, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return errors.New("无法建立存档连接池")
	}
	defer db.Close()
	app, err := poker.New(ctx, db)
	if err != nil {
		return fmt.Errorf("初始化房间存档失败: %w", err)
	}
	defer app.Close()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	number, err := strconv.Atoi(port)
	if err != nil || number < 1 || number > 65535 {
		return errors.New("PORT 必须为 1–65535")
	}
	server := &http.Server{Addr: ":" + port, Handler: app, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	finished := make(chan error, 1)
	go func() { finished <- server.ListenAndServe() }()
	log.Printf("牌桌已启动，监听端口 %s", port)
	select {
	case err := <-finished:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
		app.Close()
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
