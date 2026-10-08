package main

import (
	"context"
	"evsched/app"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	defaultConfigFile := "default.yaml"
	overrideConfigPath := flag.String("config", "", "path to override config file")
	initDBFlag := flag.Bool("init-db", false, "run River database migrations up")
	flag.Parse()

	a, err := app.New(ctx, defaultConfigFile, *overrideConfigPath, initDBFlag)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}

	if err = a.Start(ctx); err != nil {
		log.Fatalf("application error during execution or shutdown: %v", err)
	}

}
