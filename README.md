# go-sensors/asairaht10

Go library for interacting with the [ASAIR AHT10][asairaht10] and [AHT20][asairaht20] temperature and relative humidity sensors.

## Quickstart

Take a look at [rpi-sensor-exporter][rpi-sensor-exporter] for an example implementation that makes use of this sensor (and others).

[rpi-sensor-exporter]: https://github.com/go-sensors/rpi-sensor-exporter

## Sensor Details

The [ASAIR AHT10][asairaht10] and [AHT20][asairaht20] temperature and relative humidity sensors are used for temperature and relative humidity, per [vendor specifications][specs]. This [go-sensors] implementation makes use of the sensor's I2C-based protocol for obtaining measurements on an interval defined by the vendor.

[asairaht10]: http://www.aosong.com/en/products-40.html
[asairaht20]: http://www.aosong.com/en/products-32.html
[specs]: ./docs/Aosong_AHT10_en_draft_0c.pdf
[go-sensors]: https://github.com/go-sensors

## Building

This software doesn't have any compiled assets.

## Code of Conduct

We are committed to fostering an open and welcoming environment. Please read our [code of conduct](CODE_OF_CONDUCT.md) before participating in or contributing to this project.

## Contributing

We welcome contributions and collaboration on this project. Please read our [contributor's guide](CONTRIBUTING.md) to understand how best to work with us.

## License and Authors

[![Daniel James logo](https://secure.gravatar.com/avatar/eaeac922b9f3cc9fd18cb9629b9e79f6.png?size=16) Daniel James](https://github.com/thzinc)

[![license](https://img.shields.io/github/license/go-sensors/asairaht10.svg)](https://github.com/go-sensors/asairaht10/blob/master/LICENSE)
[![GitHub contributors](https://img.shields.io/github/contributors/go-sensors/asairaht10.svg)](https://github.com/go-sensors/asairaht10/graphs/contributors)

This software is made available by Daniel James under the MIT license.
