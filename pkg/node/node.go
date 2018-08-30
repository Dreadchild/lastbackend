//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2018] Last.Backend LLC
// All Rights Reserved.
//
// NOTICE:  All information contained herein is, and remains
// the property of Last.Backend LLC and its suppliers,
// if any.  The intellectual and technical concepts contained
// herein are proprietary to Last.Backend LLC
// and its suppliers and may be covered by Russian Federation and Foreign Patents,
// patents in process, and are protected by trade secret or copyright law.
// Dissemination of this information or reproduction of this material
// is strictly forbidden unless prior written permission is obtained
// from Last.Backend LLC.
//

package node

import (
	"context"
	"github.com/lastbackend/lastbackend/pkg/runtime/iri/iri"
	"os"
	"os/signal"
	"syscall"

	"github.com/lastbackend/lastbackend/pkg/node/runtime"
	"github.com/lastbackend/lastbackend/pkg/node/state"

	"github.com/lastbackend/lastbackend/pkg/api/client"
	"github.com/lastbackend/lastbackend/pkg/log"
	"github.com/lastbackend/lastbackend/pkg/node/envs"
	"github.com/lastbackend/lastbackend/pkg/node/events"
	"github.com/lastbackend/lastbackend/pkg/node/events/exporter"
	"github.com/lastbackend/lastbackend/pkg/node/http"
	"github.com/lastbackend/lastbackend/pkg/runtime/cni/cni"
	"github.com/lastbackend/lastbackend/pkg/runtime/cpi/cpi"
	"github.com/lastbackend/lastbackend/pkg/runtime/cri/cri"
	"github.com/lastbackend/lastbackend/pkg/runtime/csi/csi"
	"github.com/spf13/viper"
)

// Daemon - run node daemon
func Daemon() {

	var (
		sigs = make(chan os.Signal)
		done = make(chan bool, 1)
	)

	log.New(viper.GetInt("verbose"))
	log.Info("Start Node")

	cri, err := cri.New()
	if err != nil {
		log.Errorf("Cannot initialize cri: %v", err)
	}

	cni, err := cni.New()
	if err != nil {
		log.Errorf("Cannot initialize cni: %v", err)
	}

	cpi, err := cpi.New()
	if err != nil {
		log.Errorf("Cannot initialize cni: %v", err)
	}

	iri, err := iri.New()
	if err != nil {
		log.Errorf("Cannot initialize iri: %v", err)
	}

	csis := viper.GetStringMap("node.csi")
	if csis != nil {
		for kind := range csis {
			si, err := csi.New(kind)
			if err != nil {
				log.Errorf("Cannot initialize sni: %s > %v", kind, err)
			}
			envs.Get().SetCSI(kind, si)
		}

	}

	state := state.New()
	envs.Get().SetState(state)
	envs.Get().SetCRI(cri)
	envs.Get().SetIRI(iri)
	envs.Get().SetCNI(cni)
	envs.Get().SetCPI(cpi)

	r := runtime.NewRuntime(context.Background())
	r.Restore()

	state.Node().Info = runtime.NodeInfo()
	state.Node().Status = runtime.NodeStatus()

	cfg := client.NewConfig()

	cfg.BearerToken = viper.GetString("token")

	if viper.IsSet("api.tls") && !viper.GetBool("api.tls.insecure") {
		cfg.TLS = client.NewTLSConfig()
		cfg.TLS.CertFile = viper.GetString("api.tls.cert")
		cfg.TLS.KeyFile = viper.GetString("api.tls.key")
		cfg.TLS.CAFile = viper.GetString("api.tls.ca")
	}

	endpoint := viper.GetString("api.uri")
	rest, err := client.New(client.ClientHTTP, endpoint, cfg)
	if err != nil {
		log.Fatalf("Init client err: %s", err)
	}

	if err != nil {
		log.Errorf("node:initialize client err: %s", err.Error())
		os.Exit(0)
	}

	n := rest.V1().Cluster().Node(state.Node().Info.Hostname)
	s := rest.V1()
	envs.Get().SetClient(n, s)

	e := exporter.NewExporter()
	e.SetDispatcher(events.Dispatcher)
	envs.Get().SetExporter(e)

	if err := r.Connect(context.Background()); err != nil {
		log.Fatalf("node:initialize: connect err %s", err.Error())
	}

	r.Subscribe()

	e.Loop()
	r.Loop()

	go func() {
		opts := new(http.HttpOpts)
		opts.Insecure = viper.GetBool("node.tls.insecure")
		opts.CertFile = viper.GetString("node.tls.server_cert")
		opts.KeyFile = viper.GetString("node.tls.server_key")
		opts.CaFile = viper.GetString("node.tls.ca")

		if err := http.Listen(viper.GetString("node.host"), viper.GetInt("node.port"), opts); err != nil {
			log.Fatalf("Http server start error: %v", err)
		}
	}()

	// Handle SIGINT and SIGTERM.
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		for {
			select {
			case <-sigs:
				done <- true
				return
			}
		}
	}()

	<-done

	log.Info("Handle SIGINT and SIGTERM.")

	return
}
