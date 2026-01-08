package xid

import (
	"fmt"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"k8s.io/klog/v2"
)

type Monitor struct {
	stopCh chan any
}

func NewMonitor() *Monitor {
	return &Monitor{
		stopCh: make(chan any),
	}
}

func (m *Monitor) Start() error {
	if ret := nvml.Init(); ret != nvml.SUCCESS {
		err := fmt.Errorf("failed to initialize NVML: %s", nvml.ErrorString(ret))
		klog.Error(err.Error())
		return err
	}

	eventSet, ret := nvml.EventSetCreate()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to create event set: %v", ret)
	}
	defer func() {
		_ = eventSet.Free()
	}()

	return m.checkHealth(eventSet)
}

func (m *Monitor) Stop(_ error) {
	close(m.stopCh)

	if ret := nvml.Shutdown(); ret != nvml.SUCCESS {
		klog.Errorf("failed to shutdown NVML: %s", nvml.ErrorString(ret))
	}
}

// CheckHealth performs health checks on a set of devices, writing to the 'unhealthy' channel with any unhealthy devices
func (m *Monitor) checkHealth(eventSet nvml.EventSet) error {
	//xids := getDisabledHealthCheckXids()
	//if xids.IsAllDisabled() {
	//return nil
	//}

	//klog.Infof("Ignoring the following XIDs for health checks: %v", xids)

	uuidMap := make(map[string]bool)

	eventMask := uint64(nvml.EventTypeXidCriticalError | nvml.EventTypeDoubleBitEccError | nvml.EventTypeSingleBitEccError)

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		return fmt.Errorf("failed to get device count: %v", nvml.ErrorString(ret))
	}

	for i := 0; i < count; i++ {
		gpu, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			klog.Warningf("failed to get device at index %d: %v", i, nvml.ErrorString(ret))
			continue
		}

		uuid, ret := gpu.GetUUID()
		if ret != nvml.SUCCESS {
			klog.Warningf("failed to get uuid of device at index %d: %v", i, nvml.ErrorString(ret))
			continue
		}
		uuidMap[uuid] = true
		klog.Infof("found GPU UUID %s", uuid)

		supportedEvents, ret := gpu.GetSupportedEventTypes()
		if ret != nvml.SUCCESS {
			klog.Infof("failed to determine the supported events for %s: %s; marking it as unhealthy", uuid, nvml.ErrorString(ret))
			continue
		}

		ret = gpu.RegisterEvents(eventMask&supportedEvents, eventSet)
		if ret == nvml.ERROR_NOT_SUPPORTED {
			klog.Warningf("Device %s is too old to support healthchecking", uuid)
		}
		if ret != nvml.SUCCESS {
			klog.Infof("Marking device %v as unhealthy: %v", uuid, nvml.ErrorString(ret))
		}
	}

	for {
		select {
		case <-m.stopCh:
			return nil
		default:
		}

		e, ret := eventSet.Wait(5000)
		if ret == nvml.ERROR_TIMEOUT {
			continue
		}
		if ret != nvml.SUCCESS {
			klog.Infof("Error waiting for event: %v; Marking all devices as unhealthy", ret)
			//for _, d := range devices {
			//	unhealthy <- d
			//}
			continue
		}

		if e.EventType != nvml.EventTypeXidCriticalError {
			klog.Infof("Skipping non-nvmlEventTypeXidCriticalError event: %+v", e)
			continue
		}

		//if xids.IsDisabled(e.EventData) {
		//	klog.Infof("Skipping event %+v", e)
		//	continue
		//}

		klog.Infof("Processing event %+v", e)
		eventUUID, ret := e.Device.GetUUID()
		if ret != nvml.SUCCESS {
			// If we cannot reliably determine the device UUID, we mark all devices as unhealthy.
			klog.Infof("Failed to determine uuid for event %v: %v; Marking all devices as unhealthy.", e, ret)
			//for _, d := range devices {
			//	unhealthy <- d
			//}
			continue
		}

		if !uuidMap[eventUUID] {
			klog.Infof("Ignoring event for unexpected device: %v", eventUUID)
			continue
		}

		klog.Infof("XidCriticalError: Xid=%d on Device=%s; marking device as unhealthy.", e.EventData, eventUUID)
		//unhealthy <- d
	}
}
