// This package provides an implementation to read temperature and relative humidity measurements from an Asair AHT10 or AHT20 sensor.
//
// This implementation is based on the Adafruit AHT20 CircuitPython implementation: https://github.com/adafruit/Adafruit_CircuitPython_AHTx0
package asairaht10

import (
	"context"
	"io"
	"time"

	coreio "github.com/go-sensors/core/io"
	"github.com/go-sensors/core/units"
	"github.com/pkg/errors"
)

func reset(ctx context.Context, port coreio.Port) error {
	const cmd_reset byte = 0xBA
	_, err := port.Write([]byte{cmd_reset})
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
	case <-time.After(wakeUpTimeout):
	}

	return nil
}

type statusResponse struct {
	IsCalibrated bool
	IsBusy       bool
}

func status(port coreio.Port) (*statusResponse, error) {
	buf := make([]byte, 1)
	_, err := port.Read(buf)
	if err != nil {
		return nil, err
	}

	const calibratedMask byte = 0b00001000
	const busyMask byte = 0b10000000

	b := buf[0]
	response := &statusResponse{
		IsCalibrated: b&calibratedMask > 0,
		IsBusy:       b&busyMask > 0,
	}

	return response, nil
}

func calibrate(ctx context.Context, port coreio.Port) error {
	const cmd_calibrate byte = 0xE1
	_, err := port.Write([]byte{cmd_calibrate, 0x08, 0x00})
	if err != nil {
		return err
	}

	for {
		status, err := status(port)
		if err != nil {
			return errors.Wrap(err, "failed to read status")
		}

		if status.IsBusy {
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(statusTimeout):
			}
			continue
		}

		if !status.IsCalibrated {
			return errors.New("failed to calibrate sensor")
		}

		return nil
	}
}

func trigger(ctx context.Context, port coreio.Port) (*units.Temperature, *units.RelativeHumidity, error) {
	const cmd_trigger byte = 0xAC
	_, err := port.Write([]byte{cmd_trigger, 0x33, 0x00})
	if err != nil {
		return nil, nil, err
	}

	for {
		status, err := status(port)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to read status")
		}

		if status.IsBusy {
			select {
			case <-ctx.Done():
				return nil, nil, io.EOF
			case <-time.After(statusTimeout):
			}
			continue
		}

		buf := make([]byte, 6)
		_, err = port.Read(buf)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to read reading")
		}

		/*
		 * buf index 0       1       2       3       4       5
		 *           |-------|-------|-------|-------|-------|-------
		 * category  SSSSSSSSHHHHHHHHHHHHHHHHHHHHTTTTTTTTTTTTTTTTTTTT
		 *
		 * Categories:
		 * S: State (8 bits)
		 * H: Humidity (20 bits)
		 * T: Temperature (20 bits)
		 */

		rawHumidityReading := uint32(buf[1])<<12 | uint32(buf[2])<<4 | uint32(buf[3])>>4
		rawTemperatureReading := uint32((buf[3]&0xF))<<16 | uint32(buf[4])<<8 | uint32(buf[5])

		degreesCelsius := (float64(rawTemperatureReading) / 0x100000 * 200) - 50
		temperature := units.Temperature(degreesCelsius * float64(units.DegreeCelsius))
		percentage := float64(rawHumidityReading) / 0x100000
		relativeHumidity := units.RelativeHumidity{
			Temperature: temperature,
			Percentage:  percentage,
		}

		return &temperature, &relativeHumidity, nil
	}
}
