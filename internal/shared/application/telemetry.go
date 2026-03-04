package application

import "time"

type Telemetry interface {
	CountGateway(channelType string)
	CountRoute(route string)
	CountCommand(command string)
	CountError(route string)
	CountMemoryEvent(event string)
	ObserveDuration(route string, d time.Duration)
}
