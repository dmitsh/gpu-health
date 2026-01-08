package main

import (
	"context"
	"flag"
	"os"
	"syscall"

	"github.com/oklog/run"
	"k8s.io/klog/v2"

	"github.com/dmitsh/gpu-health/pkg/xid"
)

func main() {
	var c string
	var version bool
	flag.StringVar(&c, "c", "/etc/topograph/node-observer-config.yaml", "config file")
	flag.BoolVar(&version, "version", false, "show the version")

	klog.InitFlags(nil)
	flag.Parse()
	defer klog.Flush()

	if err := mainInternal(); err != nil {
		klog.Error(err.Error())
		os.Exit(1)
	}
}

func mainInternal() error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	xidMonitor := xid.NewMonitor()
	var g run.Group
	// Signal handler
	g.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGTERM))
	// Controller
	g.Add(xidMonitor.Start, xidMonitor.Stop)

	return g.Run()
	/*
	   ret := nvml.Init()

	   	if ret != nvml.SUCCESS {
	   		log.Fatalf("Unable to initialize NVML: %v", nvml.ErrorString(ret))
	   	}

	   	defer func() {
	   		ret := nvml.Shutdown()
	   		if ret != nvml.SUCCESS {
	   			log.Fatalf("Unable to shutdown NVML: %v", nvml.ErrorString(ret))
	   		}
	   	}()

	   count, ret := nvml.DeviceGetCount()

	   	if ret != nvml.SUCCESS {
	   		log.Fatalf("Unable to get device count: %v", nvml.ErrorString(ret))
	   	}

	   	for i := 0; i < count; i++ {
	   		device, ret := nvml.DeviceGetHandleByIndex(i)
	   		if ret != nvml.SUCCESS {
	   			log.Fatalf("Unable to get device at index %d: %v", i, nvml.ErrorString(ret))
	   		}

	   		uuid, ret := device.GetUUID()
	   		if ret != nvml.SUCCESS {
	   			log.Fatalf("Unable to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
	   		}

	   		fmt.Printf("%v\n", uuid)
	   	}
	*/
}
