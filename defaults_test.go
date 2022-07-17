package asairaht10_test

import (
	"testing"

	"github.com/go-sensors/asairaht10"
	"github.com/go-sensors/core/i2c"
	"github.com/stretchr/testify/assert"
)

func Test_GetDefaultI2CPortConfig_returns_expected_configuration(t *testing.T) {
	// Arrange
	expected := &i2c.I2CPortConfig{
		Address: 0x38,
	}

	// Act
	actual := asairaht10.GetDefaultI2CPortConfig()

	// Assert
	assert.NotNil(t, actual)
	assert.EqualValues(t, expected, actual)
}
