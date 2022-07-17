package asairaht10

import (
	"context"
	"io"
	"time"

	"github.com/go-sensors/core/humidity"
	coreio "github.com/go-sensors/core/io"
	"github.com/go-sensors/core/temperature"
	"github.com/go-sensors/core/units"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Sensor represents a configured Asair AHT10 temperature and relative humidity sensor
type Sensor struct {
	temperatures       chan *units.Temperature
	relativeHumidities chan *units.RelativeHumidity
	portFactory        coreio.PortFactory
	reconnectTimeout   time.Duration
	errorHandlerFunc   ShouldTerminate
}

// Option is a configured option that may be applied to a Sensor
type Option struct {
	apply func(*Sensor)
}

// NewSensor creates a Sensor with optional configuration
func NewSensor(portFactory coreio.PortFactory, options ...*Option) *Sensor {
	temperatures := make(chan *units.Temperature)
	relativeHumidities := make(chan *units.RelativeHumidity)
	s := &Sensor{
		temperatures:       temperatures,
		relativeHumidities: relativeHumidities,
		portFactory:        portFactory,
		reconnectTimeout:   DefaultReconnectTimeout,
		errorHandlerFunc:   nil,
	}
	for _, o := range options {
		o.apply(s)
	}
	return s
}

// WithReconnectTimeout specifies the duration to wait before reconnecting after a recoverable error
func WithReconnectTimeout(timeout time.Duration) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.reconnectTimeout = timeout
		},
	}
}

// ReconnectTimeout is the duration to wait before reconnecting after a recoverable error
func (s *Sensor) ReconnectTimeout() time.Duration {
	return s.reconnectTimeout
}

// ShouldTerminate is a function that returns a result indicating whether the Sensor should terminate after a recoverable error
type ShouldTerminate func(error) bool

// WithRecoverableErrorHandler registers a function that will be called when a recoverable error occurs
func WithRecoverableErrorHandler(f ShouldTerminate) *Option {
	return &Option{
		apply: func(s *Sensor) {
			s.errorHandlerFunc = f
		},
	}
}

// RecoverableErrorHandler a function that will be called when a recoverable error occurs
func (s *Sensor) RecoverableErrorHandler() ShouldTerminate {
	return s.errorHandlerFunc
}

const (
	wakeUpTimeout time.Duration = 20 * time.Millisecond
	statusTimeout time.Duration = 10 * time.Millisecond
)

// Run begins reading from the sensor and blocks until either an error occurs or the context is completed
func (s *Sensor) Run(ctx context.Context) error {
	defer close(s.temperatures)
	defer close(s.relativeHumidities)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(wakeUpTimeout):
		}

		port, err := s.portFactory.Open()
		if err != nil {
			return errors.Wrap(err, "failed to open port")
		}

		group, innerCtx := errgroup.WithContext(ctx)
		group.Go(func() error {
			<-innerCtx.Done()
			return port.Close()
		})
		group.Go(func() error {
			err = reset(innerCtx, port)
			if err != nil {
				return errors.Wrap(err, "failed to reset sensor")
			}

			err = calibrate(innerCtx, port)
			if err != nil {
				return errors.Wrap(err, "failed to calibrate sensor")
			}

			for {
				temperature, relativeHumidity, err := trigger(innerCtx, port)
				if err == io.EOF {
					return nil
				}
				if err != nil {
					return errors.Wrap(err, "failed to trigger reading")
				}

				select {
				case s.temperatures <- temperature:
				case <-innerCtx.Done():
					return nil
				}

				select {
				case s.relativeHumidities <- relativeHumidity:
				case <-innerCtx.Done():
					return nil
				}
			}
		})

		err = group.Wait()
		if s.errorHandlerFunc != nil {
			if s.errorHandlerFunc(err) {
				return err
			}
		}

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(s.reconnectTimeout):
		}
	}
}

// Temperatures returns a channel of temperature readings as they become available from the sensor
func (s *Sensor) Temperatures() <-chan *units.Temperature {
	return s.temperatures
}

// TemperatureSpecs returns a collection of specified measurement ranges supported by the sensor
func (*Sensor) TemperatureSpecs() []*temperature.TemperatureSpec {
	return []*temperature.TemperatureSpec{
		{
			Resolution:     10 * units.ThousandthDegreeCelsius,
			MinTemperature: -40 * units.DegreeCelsius,
			MaxTemperature: 85 * units.DegreeCelsius,
		},
	}
}

// RelativeHumidities returns a channel of relative humidity readings as they become available from the sensor
func (s *Sensor) RelativeHumidities() <-chan *units.RelativeHumidity {
	return s.relativeHumidities
}

// HumiditySpecs returns a collection of specified measurement ranges supported by the sensor
func (*Sensor) RelativeHumiditySpecs() []*humidity.RelativeHumiditySpec {
	return []*humidity.RelativeHumiditySpec{
		{
			PercentageResolution: 0.00024,
			MinPercentage:        0.0,
			MaxPercentage:        1.0,
		},
	}
}
