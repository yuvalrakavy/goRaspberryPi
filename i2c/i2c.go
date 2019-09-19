package i2c

import (
	"fmt"
	"log"
	"os"

	"golang.org/x/sys/unix"
)

const ioctlI2cSlave uint = 0x00000703

// I2Cbus Represent I2C bus
//
type I2Cbus struct {
	i2cHandle             *os.File
	lastUsedDeviceAddress byte
}

// I2Cdevice repesent a device on I2C bus
type I2Cdevice struct {
	Bus     *I2Cbus
	Address byte
}

// I2CdeviceError - Error returned from I2C device function
type I2CdeviceError struct {
	Address     byte
	Description string
}

// I2CdeviceRegisterError - Error returned from I2C device function that handle device registers
type I2CdeviceRegisterError struct {
	I2CdeviceError
	Register uint16
}

// Error - return error message
func (theError I2CdeviceError) Error() string {
	return fmt.Sprint("I2C device address ", theError.Address, ": ", theError.Description)
}

// Error - return error message
func (theError I2CdeviceRegisterError) Error() string {
	return fmt.Sprint("I2C device address ", theError.Address, " register ", theError.Register, ": ", theError.Description)

}

// Open - Open a I2C Bus device
//
func Open(unit int) (*I2Cbus, error) {
	deviceName := fmt.Sprint("/dev/i2c-", unit)

	i2cHandle, err := os.OpenFile(deviceName, os.O_RDWR, 0755)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	return &I2Cbus{i2cHandle, 0xff}, nil
}

// Close - close the bus, must be called when done with the bus (use defer)
func (bus *I2Cbus) Close() error {
	return bus.i2cHandle.Close()
}

func (bus *I2Cbus) setCurrentDeviceAddress(address byte) error {
	var err error = nil

	// Avoid set device address if it is the same as the previous
	if address != bus.lastUsedDeviceAddress {
		err = unix.IoctlSetInt(int(bus.i2cHandle.Fd()), ioctlI2cSlave, int(address))
		bus.lastUsedDeviceAddress = address
	}
	return err
}

// Device - Get device object for a given device address
func (bus *I2Cbus) Device(address byte) I2Cdevice {
	return I2Cdevice{bus, address}
}

// WriteByteRegister - Write byte value to a device's register
func (device I2Cdevice) WriteByteRegister(register uint16, value byte) error {
	if err := device.Bus.setCurrentDeviceAddress(device.Address); err != nil {
		return err
	}

	buffer := []byte{byte((register >> 8) & 0xff), byte(register & 0xff), byte(value)}
	if n, err := device.Bus.i2cHandle.Write(buffer); err != nil {
		return err
	} else if n != 3 {
		return I2CdeviceRegisterError{I2CdeviceError{device.Address, "Write byte register - write != 3"}, register}
	}

	return nil
}

// WriteWordRegister - Write 16 bit value to a device's register
func (device I2Cdevice) WriteWordRegister(register uint16, value uint16) error {
	if err := device.Bus.setCurrentDeviceAddress(device.Address); err != nil {
		return err
	}

	buffer := []byte{byte((register >> 8) & 0xff), byte(register & 0xff), byte((value >> 8) & 0xff), byte(value)}
	if n, err := device.Bus.i2cHandle.Write(buffer); err != nil {
		return err
	} else if n != 4 {
		return I2CdeviceRegisterError{I2CdeviceError{device.Address, "Write word register - write != 4"}, register}
	}

	return nil
}

// ReadByteRegister - Read byte from device's register
func (device I2Cdevice) ReadByteRegister(register uint16) (byte, error) {
	if err := device.Bus.setCurrentDeviceAddress(device.Address); err != nil {
		return 0, err
	}

	buffer := []byte{byte((register >> 8) & 0xff), byte(register & 0xff)}
	if n, err := device.Bus.i2cHandle.Write(buffer); err != nil {
		return 0, err
	} else if n != 2 {
		return 0, I2CdeviceRegisterError{I2CdeviceError{device.Address, "Read byte register - write != 2"}, register}
	}

	value := make([]byte, 1)
	if n, err := device.Bus.i2cHandle.Read(value); err != nil {
		return 0, err
	} else if n != 1 {
		return 0, I2CdeviceError{device.Address, "ReadByteRegister - did not get 1 byte as response"}
	}

	return value[0], nil
}

// ReadWordRegister - Read word (16 bits) from a device's register
func (device I2Cdevice) ReadWordRegister(register uint16) (uint16, error) {
	if err := device.Bus.setCurrentDeviceAddress(device.Address); err != nil {
		return 0, err
	}

	buffer := []byte{byte((register >> 8) & 0xff), byte(register & 0xff)}
	if n, err := device.Bus.i2cHandle.Write(buffer); err != nil {
		return 0, err
	} else if n != 2 {
		return 0, I2CdeviceRegisterError{I2CdeviceError{device.Address, "Read word register - write != 2"}, register}
	}

	value := make([]byte, 2)
	if n, err := device.Bus.i2cHandle.Read(value); err != nil {
		return 0, err
	} else if n != 2 {
		return 0, I2CdeviceError{device.Address, "ReadWordRegister - did not get 2 bytes as response"}
	}

	return (uint16(value[0]) << 8) | uint16(value[1]), nil

}
