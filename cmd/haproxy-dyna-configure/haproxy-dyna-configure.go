package main

import (
	"context"
	"os"

	"github.com/openshift-splat-team/haproxy-dyna-configure/pkg"
	controller "github.com/openshift-splat-team/haproxy-dyna-configure/pkg/controller"
	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.TODO()
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
	err := pkg.Initialize(ctx)
	if err != nil {
		log.Errorf("unable to initialize %s", err)
		return
	}

	controller.StartManager()
	/*	cfg, err := pkg.CheckRanges(ctx)
		if err != nil {
			log.Errorf("unable to check ranges %s", err)
			return
		}*/
	/*err = pkg.BuildDynamicConfiguration(cfg)
	if err != nil {
		log.Errorf("unable to apply configuration %s", err)
		return
	}*/
}
