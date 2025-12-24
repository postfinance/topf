package topf

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/cosi-project/runtime/pkg/safe"
	"github.com/siderolabs/go-retry/retry"
	"github.com/siderolabs/talos/pkg/machinery/api/machine"
	"github.com/siderolabs/talos/pkg/machinery/client"
	"github.com/siderolabs/talos/pkg/machinery/resources/runtime"
)

// Stabilize waits for the talos machine to be stable (running and ready) for a given duration
func (n *Node) Stabilize(ctx context.Context, logger *slog.Logger, stabilizationDuration time.Duration) error { //nolint:gocognit // TODO: refactor
	// machine status covers apid, machined, kubelet, etcd, trustd, time, network, service, staticPods, nodeReady
	waitForMachineReady := func(ctx context.Context) error {
		// immediately start watching for machine status events such that we don't miss those between
		// after initial check and starting to watch
		eventChannel := make(chan client.EventResult)

		nodeClient, err := n.Client(ctx)
		if err != nil {
			return retry.ExpectedErrorf("couldn't get client: %w", err)
		}
		defer nodeClient.Close()

		err = nodeClient.EventsWatchV2(ctx, eventChannel)
		if err != nil {
			return retry.ExpectedErrorf("couldn't watch events: %w", err)
		}

		machineStatus, err := safe.ReaderGetByID[*runtime.MachineStatus](ctx, nodeClient.COSI, runtime.MachineStatusID)
		if err != nil {
			return retry.ExpectedErrorf("couldn't get machine status: %s", machineStatus.TypedSpec().Stage)
		}

		if machineStatus.TypedSpec().Stage != runtime.MachineStageRunning {
			return retry.ExpectedErrorf("machine not in stage running: %s", machineStatus.TypedSpec().Stage)
		}

		if !machineStatus.TypedSpec().Status.Ready {
			reasons := []string{}
			for _, cond := range machineStatus.TypedSpec().Status.UnmetConditions {
				reasons = append(reasons, fmt.Sprintf("%s: %s", cond.Name, cond.Reason))
			}

			return retry.ExpectedErrorf("machine not ready (%v)", reasons)
		}

		// we're healthy but we wait for another 30 seconds, to see if nothing bad happens
		stabilizationDeadline := time.After(stabilizationDuration)

		logger.Info("machine ready, waiting for stabilization...", "duration", stabilizationDuration)

		for {
			select {
			case <-stabilizationDeadline:
				// nothing bad happened
				return nil
			case e := <-eventChannel:
				if e.Error != nil {
					return retry.ExpectedErrorf("error event received: %w", e.Error)
				}

				// we're mainly interested in machine status events
				switch msg := e.Event.Payload.(type) {
				case *machine.MachineStatusEvent:
					if msg.GetStage() != machine.MachineStatusEvent_RUNNING {
						return retry.ExpectedErrorf("machine not in stage running: %s", msg.GetStage())
					}

					if !msg.GetStatus().GetReady() {
						reasons := []string{}
						for _, cond := range machineStatus.TypedSpec().Status.UnmetConditions {
							reasons = append(reasons, fmt.Sprintf("%s: %s", cond.Name, cond.Reason))
						}

						return retry.ExpectedErrorf("machine not ready (%v)", reasons)
					}
				default:
					// just for debugging we show unrelated events
					logger.Debug("event", "type", e.Event.TypeURL, "payload", e.Event.Payload)
				}
			}
		}
	}

	return retry.Constant(time.Minute*15, retry.WithErrorLogging(logger.Enabled(ctx, slog.LevelDebug))).RetryWithContext(ctx, waitForMachineReady)
}
