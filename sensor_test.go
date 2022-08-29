package asairaht10_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-sensors/asairaht10"
	"github.com/go-sensors/core/humidity"
	"github.com/go-sensors/core/io/mocks"
	"github.com/go-sensors/core/temperature"
	"github.com/go-sensors/core/units"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

func Test_NewSensor_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	// Act
	sensor := asairaht10.NewSensor(portFactory)

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, asairaht10.DefaultReconnectTimeout, sensor.ReconnectTimeout())
	assert.Nil(t, sensor.RecoverableErrorHandler())
}

func Test_NewSensor_with_options_returns_a_configured_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	expectedReconnectTimeout := asairaht10.DefaultReconnectTimeout * 10

	// Act
	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithReconnectTimeout(expectedReconnectTimeout),
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	// Assert
	assert.NotNil(t, sensor)
	assert.Equal(t, expectedReconnectTimeout, sensor.ReconnectTimeout())
	assert.NotNil(t, sensor.RecoverableErrorHandler())
	assert.True(t, sensor.RecoverableErrorHandler()(nil))
}

func Test_TemperatureSpecs_returns_supported_temperatures(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	sensor := asairaht10.NewSensor(portFactory)
	expected := []*temperature.TemperatureSpec{
		{
			Resolution:     10 * units.ThousandthDegreeCelsius,
			MinTemperature: -40 * units.DegreeCelsius,
			MaxTemperature: 85 * units.DegreeCelsius,
		},
	}

	// Act
	actual := sensor.TemperatureSpecs()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}

func Test_RelativeHumiditySpecs_returns_supported_relative_humidities(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	sensor := asairaht10.NewSensor(portFactory)
	expected := []*humidity.RelativeHumiditySpec{
		{
			PercentageResolution: 0.00024,
			MinPercentage:        0.0,
			MaxPercentage:        1.0,
		},
	}

	// Act
	actual := sensor.RelativeHumiditySpecs()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}

func Test_Run_fails_when_opening_port(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(nil, errors.New("boom"))
	sensor := asairaht10.NewSensor(portFactory)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to open port")
}

func Test_Run_fails_to_reset_sensor(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to reset sensor")
}

func Test_Run_fails_to_calibrate_sensor_when_issuing_command(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to calibrate sensor")
}

func Test_Run_fails_to_calibrate_sensor_while_reading_status(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		Return(0, nil)
	busyStatusResponse := port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b10000000
			return len(buf), nil
		})
	port.EXPECT().
		Read(gomock.Any()).
		After(busyStatusResponse).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to calibrate sensor")
}

func Test_Run_fails_to_calibrate_sensor_when_the_sensor_is_not_calibrated(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		Return(0, nil)
	port.EXPECT().
		Read(gomock.Any()).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00000000
			return len(buf), nil
		})
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to calibrate sensor")
}

func Test_Run_fails_to_trigger_sensor_when_issuing_command(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	reset := port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	calibrate := port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		After(reset).
		Return(0, nil)
	calibratedStatus := port.EXPECT().
		Read(gomock.Any()).
		After(calibrate).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	port.EXPECT().
		Write([]byte{0xAC, 0x33, 0x00}).
		After(calibratedStatus).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to trigger reading")
}

func Test_Run_fails_to_trigger_sensor_when_reading_status(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	reset := port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	calibrate := port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		After(reset).
		Return(0, nil)
	calibratedStatus := port.EXPECT().
		Read(gomock.Any()).
		After(calibrate).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	trigger := port.EXPECT().
		Write([]byte{0xAC, 0x33, 0x00}).
		After(calibratedStatus).
		Return(0, nil)
	busyStatus := port.EXPECT().
		Read(gomock.Any()).
		After(trigger).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b10000000
			return len(buf), nil
		})
	port.EXPECT().
		Read(gomock.Any()).
		After(busyStatus).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to trigger reading")
	assert.ErrorContains(t, err, "failed to read status")
}

func Test_Run_fails_to_trigger_sensor_when_reading_measurement(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	reset := port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	calibrate := port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		After(reset).
		Return(0, nil)
	calibratedStatus := port.EXPECT().
		Read(gomock.Any()).
		After(calibrate).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	trigger := port.EXPECT().
		Write([]byte{0xAC, 0x33, 0x00}).
		After(calibratedStatus).
		Return(0, nil)
	readyStatus := port.EXPECT().
		Read(gomock.Any()).
		After(trigger).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	port.EXPECT().
		Read(gomock.Any()).
		After(readyStatus).
		Return(0, errors.New("boom"))
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.ErrorContains(t, err, "failed to trigger reading")
	assert.ErrorContains(t, err, "failed to read reading")
}

func Test_Run_returns_expected_measurements(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)
	portFactory := mocks.NewMockPortFactory(ctrl)

	port := mocks.NewMockPort(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	reset := port.EXPECT().
		Write([]byte{0xBA}).
		Return(0, nil)
	calibrate := port.EXPECT().
		Write([]byte{0xE1, 0x08, 0x00}).
		After(reset).
		Return(0, nil)
	calibratedStatus := port.EXPECT().
		Read(gomock.Any()).
		After(calibrate).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	trigger := port.EXPECT().
		Write([]byte{0xAC, 0x33, 0x00}).
		After(calibratedStatus).
		Return(0, nil).
		AnyTimes()
	readyStatus := port.EXPECT().
		Read(gomock.Any()).
		After(trigger).
		DoAndReturn(func(buf []byte) (int, error) {
			buf[0] = 0b00001000
			return len(buf), nil
		})
	expectedHumidityPercentage := float64(0.55)
	expectedTemperature := 50 * units.DegreeCelsius
	validReading := port.EXPECT().
		Read(gomock.Any()).
		After(readyStatus).
		DoAndReturn(func(buf []byte) (int, error) {
			rawHumidity := uint32(expectedHumidityPercentage * 0x100000)
			rawTemperature := uint32(((expectedTemperature.DegreesCelsius() + 50) * 0x100000 / 200))
			buf[0] = 0b00001000
			buf[1] = byte((rawHumidity >> 12) & 0xFF)
			buf[2] = byte((rawHumidity >> 4) & 0xFF)
			buf[3] = byte((rawHumidity<<4)&0xF0) | byte((rawTemperature>>16)&0x0F)
			buf[4] = byte((rawTemperature >> 8) & 0xFF)
			buf[5] = byte((rawTemperature) & 0xFF)

			return len(buf), nil
		})
	{ // Set up mock to accept another trigger command and then return busy status
		port.EXPECT().
			Write([]byte{0xAC, 0x33, 0x00}). // Trigger
			After(validReading).
			Return(0, nil)
		port.EXPECT().
			Read(gomock.Any()).
			After(trigger).
			DoAndReturn(func(buf []byte) (int, error) {
				buf[0] = 0b10001000 // Busy (and calibrated)
				return len(buf), nil
			})
	}
	port.EXPECT().
		Close().
		Return(nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return true }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	group.Go(func() error {
		select {
		case actualTemperature, ok := <-sensor.Temperatures():
			assert.True(t, ok)
			assert.NotNil(t, actualTemperature)
			assert.Equal(t, expectedTemperature, *actualTemperature)
		case <-time.After(3 * time.Second):
			assert.Fail(t, "failed to receive temperature in expected amount of time")
		}

		select {
		case actualHumidity, ok := <-sensor.RelativeHumidities():
			assert.True(t, ok)
			assert.NotNil(t, actualHumidity)
			assert.InEpsilon(t, expectedHumidityPercentage, actualHumidity.Percentage, 0.00024)
			assert.Equal(t, expectedTemperature, actualHumidity.Temperature)
		case <-time.After(3 * time.Second):
			assert.Fail(t, "failed to receive temperature in expected amount of time")
		}

		cancel()
		return nil
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
}

func Test_Run_attempts_to_recover_from_failure(t *testing.T) {
	// Arrange
	ctrl := gomock.NewController(t)

	port := mocks.NewMockPort(ctrl)
	port.EXPECT().
		Write(gomock.Any()).
		Return(0, errors.New("boom")).
		AnyTimes()
	port.EXPECT().
		Close().
		Times(1)

	portFactory := mocks.NewMockPortFactory(ctrl)
	portFactory.EXPECT().
		Open().
		Return(port, nil)

	sensor := asairaht10.NewSensor(portFactory,
		asairaht10.WithRecoverableErrorHandler(func(err error) bool { return false }))

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	group, ctx := errgroup.WithContext(ctx)

	// Act
	group.Go(func() error {
		return sensor.Run(ctx)
	})
	err := group.Wait()

	// Assert
	assert.Nil(t, err)
}
