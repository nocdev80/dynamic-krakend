package main

import (
	"flag"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luraproject/lura/config"
	"github.com/luraproject/lura/logging"
	"github.com/luraproject/lura/proxy"
	"github.com/luraproject/lura/router"
	krakendgin "github.com/luraproject/lura/router/gin"
	notify "github.com/nocdev80/go-notify"
)

var routerFactory router.Factory

func main() {
	port := flag.Int("p", 9091, "Port of the service")
	logLevel := flag.String("l", "ERROR", "Logging level")
	debug := flag.Bool("debug", false, "Enable the debug")
	directory := flag.String("d", "./", "Directory")
	configFile := flag.String("c", "configuration.json", "Path to the configuration filename")
	flag.Parse()

	parser := config.NewParser()
	serviceConfig, err := parser.Parse(*configFile)
	if err != nil {
		log.Fatal("ERROR:", err)
	}
	serviceConfig.Debug = serviceConfig.Debug || *debug
	if *port != 0 {
		serviceConfig.Port = *port
	}

	logger, err := logging.NewLogger(*logLevel, os.Stdout, "[KRAKEND]")
	if err != nil {
		log.Fatal("ERROR:", err.Error())
	}

	// routerFactory := krakendgin.DefaultFactory(proxy.DefaultFactory(logger), logger)

	routerFactory = krakendgin.NewFactory(krakendgin.Config{
		Engine:       gin.Default(),
		ProxyFactory: customProxyFactory{logger, proxy.DefaultFactory(logger)},
		Logger:       logger,
		HandlerFactory: func(configuration *config.EndpointConfig, proxy proxy.Proxy) gin.HandlerFunc {
			return krakendgin.EndpointHandler(configuration, proxy)
		},
		RunServer: router.RunServer,
	})

	r1 := routerFactory.New()

	update := func(observer *notify.ObserverNotify, event *notify.Event) {
		parser := config.NewParser()
		time.Sleep(time.Millisecond * 500)
		serviceConfig, err := parser.Parse(observer.Filename)
		if err != nil {
			log.Println("ERROR:", err)
			return
		}
		r1.ResteEngine()
		r1.RegisterKrakendEndpoints(serviceConfig)
		log.Println("INFO:", "configuration reloaded")
	}

	notify.NewObserverNotify(*directory, *configFile).
		FxCreate(update).
		FxWrite(update).
		FxChmod(update).
		//FxRename(update).
		//FxRemove(update).
		Run()
	r1.Run(serviceConfig)
}

//
// customProxyFactory adds a logging middleware wrapping the internal factory
type customProxyFactory struct {
	logger  logging.Logger
	factory proxy.Factory
}

// New implements the Factory interface
func (cf customProxyFactory) New(cfg *config.EndpointConfig) (p proxy.Proxy, err error) {
	p, err = cf.factory.New(cfg)
	if err == nil {
		p = proxy.NewLoggingMiddleware(cf.logger, cfg.Endpoint)(p)
	}
	return
}
