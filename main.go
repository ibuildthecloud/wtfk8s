package main

import (
	"flag"
	"io"
	"log"
	"os"

	"github.com/ibuildthecloud/wtfk8s/pkg/differ"
	"github.com/ibuildthecloud/wtfk8s/pkg/watcher"
	"github.com/rancher/wrangler/pkg/clients"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"k8s.io/klog/v2"
)

var (
	kubeConfig = flag.String("kubeconfig", "", "Kube config location")
	context    = flag.String("context", "", "Kube config context")
	namespace  = flag.String("namespace", "", "Limit to namespace")
)

func main() {
	flag.Parse()
	if err := mainErr(); err != nil {
		log.Fatalln(err)
	}
}

func mainErr() error {
	klog.SetOutput(io.Discard)
	logrus.SetLevel(logrus.FatalLevel)

	ctx := signals.SetupSignalContext()
	restConfig := kubeconfig.GetNonInteractiveClientConfigWithContext(*kubeConfig, *context)

	clients, err := clients.New(restConfig, &generic.FactoryOptions{
		Namespace: *namespace,
	})
	if err != nil {
		return err
	}

	watcher, err := watcher.New(clients)
	if err != nil {
		return err
	}

	for _, arg := range os.Args[1:] {
		watcher.MatchName(arg)
	}

	differ, err := differ.New(clients)
	if err != nil {
		return err
	}

	objs, err := watcher.Start(ctx)
	if err != nil {
		return err
	}

	go func() {
		for obj := range objs {
			_ = differ.Print(obj)
		}
	}()

	if err := clients.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()
	return nil
}
