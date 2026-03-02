package application

import "time"

type Telemetry interface {
	CountGateway(source string)
	CountRoute(route string)
	CountCommand(command string)
	CountError(route string)
	ObserveDuration(route string, d time.Duration)
}
