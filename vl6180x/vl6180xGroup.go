package vl6180x

import (
	"fmt"
	"time"

	"github.com/yuvalrakavy/goPool"
	"github.com/yuvalrakavy/goRaspberryPi/i2c"
)

const defaultVl6180xAddress = 41
const sensorBootTimeMs = 400

type Vl6180xGroup []Vl6180x

type RangeValueMessage struct {
	Sensor   Vl6180x
	Distance byte
}

// ScanBus - return group of all VL6180x sensors found on the bus
//
func ScanBus(bus *i2c.I2Cbus) (Vl6180xGroup, error) {
	sensors := make([]Vl6180x, 0, 10)

	for address := byte(0); address < 127; address++ {
		if IsVL6180x(bus, address) == nil {
			sensors = append(sensors, Device(bus, address))
		}
	}

	return sensors, nil
}

// AssignAddresses - find all vl6180x sensors on the i2c bus. Assign address to each one.
//
// In additon to power and i2c bus which is connected in parallel to all the VL6180x, the sensors are chained so
// the GPIO1 output is connected to the GPIO0/CE input of the next VL6180x. It is assumed that a controller's GPIO output
// pin is connected to the GPIO0/CE of the chain's first VL6180x.
//
//    ______________              _____________           _____________           _____________
//    | Controller |              | VL6180x 1 |           | VL6180x 2 |           | VL6180x 3 |
//    |            |              |       __  |           |       __  |           |       __  |
//    |   GPIO out |______________| GPIO0/CE  |       ____| GPIO0/CE  |       ____| GPIO0/CE  |
//    |            |              |           |      /    |           |      /    |           |
//    |            |              |     GPIO1 |_____/     |     GPIO1 |_____/     |     GPIO1 |
//    |____________|              |___________|           |___________|           |___________|
//
// The process is as follows:
//
//   Place the controller GPIO pin low, this will place the first VL6180X in the chain in reset state
//   Wait a bit, and put the controller's GPIO pin high, taking the VL6180x out of reset. Wait for the
//   chip to initialize.
//
//   Repeat the following as long as there is a valid VL6180X on the bus at the default address (41):
//       1. Set the VL6180x at the default address GPIO1 to low - placing the next VL6180x in the chain in reset
//       2. Initialize the VL6180x at the default address (41)
//       3. Change the address of the VL6180x from the default (41) to address N.
//       4. Set the GPIO1 pin of the VL6180x at address N to high, taking the next VL6180x in the chain out
//          of reset state
//       5. Wait for the VL6180x just taken out of reset to initialize
//       6. Set N to N+1
//
//   Parameters:
//      bus - The i2c bus to scan
//      startAddress - The address to assign to first sensor in the chain
//      resetStateOn - function that would place the first sensor in the chain in reset state
//      reserStateOff - function that would take the first sensor in the chain out of reset state
//
func AssignAddresses(bus *i2c.I2Cbus, startAddress byte, resetStateOn func(), resetStateOff func()) (Vl6180xGroup, error) {
	sensors := make(Vl6180xGroup, 0, 10)
	address := startAddress
	var sensor *Vl6180x = nil

	for {
		// Place next sensor in reset state
		if sensor == nil {
			resetStateOn()
		} else {
			sensor.SetGPIO1low() // Put next sensor in reset state
		}

		time.Sleep(10 * time.Millisecond)

		// Take sensor out of reset state
		if sensor == nil {
			resetStateOff()
		} else {
			sensor.SetGPIO1high() // Take sensor out of reset
		}

		// Allow the sensor to boot
		time.Sleep(sensorBootTimeMs * time.Millisecond)

		// Check if there is a sensor at the default address
		if err := IsVL6180x(bus, byte(defaultVl6180xAddress)); err != nil {
			break
		}

		// Found sensor at the default address
		nextSensor := Device(bus, defaultVl6180xAddress)
		if err := nextSensor.Initialize(); err != nil {
			return sensors, err
		}

		nextSensor.SetAddress(address)
		address = address + 1

		sensors = append(sensors, nextSensor)
		sensor = &nextSensor
	}

	return sensors, nil
}

func (sensors Vl6180xGroup) Initialize() error {
	for _, sensor := range sensors {
		if err := sensor.Initialize(); err != nil {
			return err
		}
	}

	return nil
}

// GetRangeReadingChannel - Get a channel the will receive range reading messages from all the sensors
// in the group. The reading process is initialized. It will terminate when the pool is terminated
//
func (sensors Vl6180xGroup) GetRangeReadingChannel(pool *goPool.GoPool) (*goPool.GoPool, <-chan RangeValueMessage) {
	valuesChannel := make(chan RangeValueMessage, len(sensors))

	go func() {
		currentValues := make(map[byte]byte)
		pool.Enter()
		defer pool.Leave()
		defer close(valuesChannel)

		// Put sensors in continuous range reading mode
		for _, sensor := range sensors {
			if err := sensor.StartRangeContinuous(100); err != nil {
				panic(err)
			}
		}

		defer func() {
			// End continuous mode
			fmt.Println("StopContinuous")
			for _, sensor := range sensors {
				if err := sensor.StopContinuous(); err != nil {
					panic(err)
				}
			}
		}()

		time.Sleep(100 * time.Millisecond)

		for {
			for _, sensor := range sensors {
				select {
				case <-pool.Done:
					fmt.Println("GetrangeReadingChannel terminated")
					return

				default:
					if valueAvailable, value, err := sensor.PeekRange(); err == nil {
						if valueAvailable {
							currentValue, hasCurrentValue := currentValues[sensor.Address]

							if !hasCurrentValue || currentValue != value {
								currentValues[sensor.Address] = value
								valuesChannel <- RangeValueMessage{Sensor: sensor, Distance: value}
							}
						}
					} else {
						panic(err)
					}
				}
			}
		}
	}()

	return pool, valuesChannel
}
