package vl6180x

import (
	"fmt"
	"time"

	"github.com/yuvalrakavy/goPool"
	"github.com/yuvalrakavy/goRaspberryPi/i2c"
)

type Vl6180xGroup []Vl6180x

type RangeValueMessage struct {
	Sensor   Vl6180x
	Distance byte
}

func ScanBus(bus *i2c.I2Cbus) (Vl6180xGroup, error) {
	sensors := make([]Vl6180x, 0, 10)

	for address := byte(0); address < 127; address++ {
		if IsVL6180x(bus, address) == nil {
			sensors = append(sensors, Device(bus, address))
		}
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
