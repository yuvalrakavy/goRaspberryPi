package vl6180x

import (
	"fmt"
	"time"

	"github.com/yuvalrakavy/goRaspberryPi/i2c"
)

// VL6180 control registers
// nolint:go-lint,varcheck,deadcode
const (
	registerIdentificationModelID        = 0x000
	registerIdentificationModelRevMajor  = 0x001
	registerIdentificationModelRevMinor  = 0x002
	registerIdentificationModuleRevMajor = 0x003
	registerIdentificationModuleRevMinor = 0x004
	registerIdentificationDateHi         = 0x006
	registerIdentificationDateLo         = 0x007
	registerIdentificationTime           = 0x008 // 16-bit

	registerSystemModeGpio0            = 0x010
	registerSystemModeGpio1            = 0x011
	registerSystemHistoryCtrl          = 0x012
	registerSystemInterruptConfigGpio  = 0x014
	registerSystemInterruptClear       = 0x015
	registerSystemFreshOutOfReset      = 0x016
	registerSystemGroupedParameterHold = 0x017

	registerSysrangeStart                     = 0x018
	registerSysrangeThreshHigh                = 0x019
	registerSysrangeThreshLow                 = 0x01A
	registerSysrangeIntermeasurementPeriod    = 0x01B
	registerSysrangeMaxConvergenceTime        = 0x01C
	registerSysrangeCrosstalkCompensationRate = 0x01E // 16-bit
	registerSysrangeCrosstalkValidHeight      = 0x021
	registerSysrangeEarlyConvergenceEstimate  = 0x022 // 16-bit
	registerSysrangePartToPartRangeOffset     = 0x024
	registerSysrangeRangeIgnoreValidHeight    = 0x025
	registerSysrangeRangeIgnoreThreshold      = 0x026 // 16-bit
	registerSysrangeMaxAmbientLevelMult       = 0x02C
	registerSysrangeRangeCheckEnables         = 0x02D
	registerSysrangeVhvRecalibrate            = 0x02E
	registerSysrangeVhvRepeatRate             = 0x031

	registerSysalsStart                  = 0x038
	registerSysalsThreshHigh             = 0x03A
	registerSysalsThreshLow              = 0x03C
	registerSysalsIntermeasurementPeriod = 0x03E
	registerSysalsAnalogueGain           = 0x03F
	registerSysalsIntegrationPeriod      = 0x040

	registerResultRangeStatus               = 0x04D
	registerResultAlsStatus                 = 0x04E
	registerResultInterruptStatusGpio       = 0x04F
	registerResultAlsVal                    = 0x050 // 16-bit
	registerResultHistoryBuffer0            = 0x052 // 16-bit
	registerResultHistoryBuffer1            = 0x054 // 16-bit
	registerResultHistoryBuffer2            = 0x056 // 16-bit
	registerResultHistoryBuffer3            = 0x058 // 16-bit
	registerResultHistoryBuffer4            = 0x05A // 16-bit
	registerResultHistoryBuffer5            = 0x05C // 16-bit
	registerResultHistoryBuffer6            = 0x05E // 16-bit
	registerResultHistoryBuffer7            = 0x060 // 16-bit
	registerResultRangeVal                  = 0x062
	registerResultRangeRaw                  = 0x064
	registerResultRangeReturnRate           = 0x066 // 16-bit
	registerResultRangeReferenceRate        = 0x068 // 16-bit
	registerResultRangeReturnSignalCount    = 0x06C // 32-bit
	registerResultRangeReferenceSignalCount = 0x070 // 32-bit
	registerResultRangeReturnAmbCount       = 0x074 // 32-bit
	registerResultRangeReferenceAmbCount    = 0x078 // 32-bit
	registerResultRangeReturnConvTime       = 0x07C // 32-bit
	registerResultRangeReferenceConvTime    = 0x080 // 32-bit

	registerRangeScaler = 0x096 // 16-bit - see STSW-IMG003 core/inc/vl6180x_def.h

	registerReadoutAveragingSamplePeriod = 0x10A
	registerFirmwareBootup               = 0x119
	registerFirmwareResultScaler         = 0x120
	registerI2CSlaveDeviceAddress        = 0x212
	registerInterleavedModeEnable        = 0x2A3
)

// Vl6180x - ST Electronics time of flight sensor
type Vl6180x struct {
	i2c.I2Cdevice
}

// Timeout error is returned on read timeout
type Timeout struct {
	i2c.I2CdeviceError
}

type Vl6180identification struct {
	Model          byte
	ModelRevMajor  byte
	ModelRevMinor  byte
	ModuleRevMajor byte
	ModuleRevMinor byte
	Date           uint16
	Time           uint16
}

// Device - get Vl6180x device at a given address
func Device(bus *i2c.I2Cbus, address byte) Vl6180x {
	return Vl6180x{bus.Device(address)}
}

// IsVL6180x return true if the device at a given I2C bus address is a VL6180x
func IsVL6180x(bus *i2c.I2Cbus, address byte) error {
	var value byte
	var err error

	if value, err = bus.Device(address).ReadByteRegister(registerSystemFreshOutOfReset); err != nil {
		return err
	} else if value != 1 {
		return i2c.I2CdeviceRegisterError{I2CdeviceError: i2c.I2CdeviceError{Address: address, Description: fmt.Sprintf("Expected 1 got %x", value)}, Register: registerSystemFreshOutOfReset}
	}

	return nil
}

type registerSettingsTable []struct {
	register uint16
	value    byte
}

func (device Vl6180x) setRegisters(settingTable registerSettingsTable) error {
	for _, entry := range settingTable {
		if err := device.WriteByteRegister(entry.register, entry.value); err != nil {
			return err
		}
	}

	return nil
}

// Initialize - initialize device for proper operation.
func (device Vl6180x) Initialize() error {
	if err := IsVL6180x(device.Bus, device.Address); err != nil {
		return err
	}

	initializationValues := registerSettingsTable{
		{0x0207, 0x01},
		{0x0208, 0x01},
		{0x0096, 0x00},
		{0x0097, 0xfd},
		{0x00e3, 0x00},
		{0x00e4, 0x04},
		{0x00e5, 0x02},
		{0x00e6, 0x01},
		{0x00e7, 0x03},
		{0x00f5, 0x02},
		{0x00d9, 0x05},
		{0x00db, 0xce},
		{0x00dc, 0x03},
		{0x00dd, 0xf8},
		{0x009f, 0x00},
		{0x00a3, 0x3c},
		{0x00b7, 0x00},
		{0x00bb, 0x3c},
		{0x00b2, 0x09},
		{0x00ca, 0x09},
		{0x0198, 0x01},
		{0x01b0, 0x17},
		{0x01ad, 0x00},
		{0x00ff, 0x05},
		{0x0100, 0x05},
		{0x0199, 0x05},
		{0x01a6, 0x1b},
		{0x01ac, 0x3e},
		{0x01a7, 0x1f},
		{0x0030, 0x00},

		{registerSystemGroupedParameterHold, 1},

		{registerReadoutAveragingSamplePeriod, 48},
		{registerSysalsAnalogueGain, 0x46},             // ALS gain = 1 nominal, actually 1.01 according to Table 14 in datasheet
		{registerSysrangeVhvRepeatRate, 0xff},          // auto Very High Voltage temperature recalibration after every 255 range measurements
		{registerSysrangeVhvRecalibrate, 0x01},         // Manually trigger recalibration
		{registerSysrangeIntermeasurementPeriod, 0x09}, // 100ms
		{registerSysalsIntermeasurementPeriod, 0x31},   // 500ms
		{registerSystemInterruptConfigGpio, 0x24},      // ALS new sample ready interrupt); range_int_mode = 4 (range new sample ready interrupt
		{registerSysrangeMaxConvergenceTime, 0x31},     // 49ms
		{registerInterleavedModeEnable, 0x00},          // disable interleaved mode
		{registerSystemInterruptClear, 0x7},            // clear all interrupts
		{registerSysrangeStart, 0x00},                  // Clear range start register

		{registerSystemGroupedParameterHold, 0},
	}

	if err := device.setRegisters(initializationValues); err != nil {
		return err
	}

	if err := device.WriteWordRegister(registerSysalsIntegrationPeriod, 0x0063); err != nil {
		return err
	}

	if err := device.SetScaling(1); err != nil {
		return err
	}

	return nil
}

// GetIdentification - get device information
func (device Vl6180x) GetIdentification() (*Vl6180identification, error) {
	var result Vl6180identification
	var err error

	if result.Model, err = device.ReadByteRegister(registerIdentificationModelID); err != nil {
		return nil, err
	}

	if result.ModelRevMajor, err = device.ReadByteRegister(registerIdentificationModelRevMajor); err != nil {
		return nil, err
	}

	if result.ModelRevMinor, err = device.ReadByteRegister(registerIdentificationModelRevMinor); err != nil {
		return nil, err
	}

	if result.ModuleRevMajor, err = device.ReadByteRegister(registerIdentificationModuleRevMajor); err != nil {
		return nil, err
	}

	if result.ModuleRevMinor, err = device.ReadByteRegister(registerIdentificationModuleRevMinor); err != nil {
		return nil, err
	}

	if result.Date, err = device.ReadWordRegister(registerIdentificationDateHi); err != nil {
		return nil, err
	}

	if result.Time, err = device.ReadWordRegister(registerIdentificationTime); err != nil {
		return nil, err
	}

	return &result, nil
}

// SetAddress - change the device address on the bus
func (device *Vl6180x) SetAddress(newAddress byte) error {
	if err := device.WriteByteRegister(registerI2CSlaveDeviceAddress, newAddress); err != nil {
		return err
	}

	device.Address = newAddress
	return nil
}

// SetScaling - set measurment scale factor
//
// Set range scaling factor. The sensor uses 1x scaling by default, giving range
// measurements in units of mm. Increasing the scaling to 2x or 3x makes it give
// raw values in units of 2 mm or 3 mm instead. In other words, a bigger scaling
// factor increases the sensor's potential maximum range but reduces its
// resolution.
func (device Vl6180x) SetScaling(scale byte) error {
	var err error
	var partToPartRangeOffset byte
	const defaultCrosstalkValidHeight = 20

	scalerValues := []uint16{0, 253, 127, 84}

	if scale < 1 || scale > 3 {
		return i2c.I2CdeviceError{Address: device.Address, Description: "Invalid scale factor (not between 1...3)"}
	}

	if partToPartRangeOffset, err = device.ReadByteRegister(registerSysrangePartToPartRangeOffset); err != nil {
		return err
	}

	if err = device.WriteWordRegister(registerRangeScaler, scalerValues[scale]); err != nil {
		return err
	}

	if err = device.WriteByteRegister(registerSysrangePartToPartRangeOffset, partToPartRangeOffset/scale); err != nil {
		return err
	}

	if err = device.WriteByteRegister(registerSysrangeCrosstalkValidHeight, defaultCrosstalkValidHeight/scale); err != nil {
		return err
	}

	if earlyConvergenceEnabled, err := device.ReadByteRegister(registerSysrangeRangeCheckEnables); err == nil {
		var mask byte

		if scale == 1 {
			mask = 1
		}

		if err = device.WriteByteRegister(registerSysrangeRangeCheckEnables, (earlyConvergenceEnabled&0xf)|mask); err != nil {
			return err
		}
	} else {
		return err
	}

	return nil
}

// ReadRange - Performs a single-shot ranging measurement
//  if timeout != 0, wait upto timeout millseconds for reading
func (device Vl6180x) ReadRange(timeout int) (byte, error) {
	if err := device.WriteByteRegister(registerSysrangeStart, 0x01); err != nil {
		return 0xff, err
	}

	return device.ReadRangeContinous(timeout)
}

// ReadAmbient - Performs a single-shot ambient measurement
//  if timeout != 0, wait upto timeout millseconds for reading
func (device Vl6180x) ReadAmbient(timeout int) (uint16, error) {
	if err := device.WriteByteRegister(registerSysalsStart, 0x01); err != nil {
		return 0, err
	}

	return device.ReadAmbientContinous(timeout)
}

// VStartRangeContinuous - Starts continuous ranging measurements with the given period in ms
// (10 ms resolution; defaults to 100 ms if not specified).
//
// The period must be greater than the time it takes to perform a
// measurement. See section 2.4.4 ("Continuous mode limits") in the datasheet
// for details.
func (device Vl6180x) StartRangeContinuous(period uint16) error {
	periodRegisterValue := uint16(period/10 - 1)

	if periodRegisterValue > 254 {
		periodRegisterValue = 254
	}

	if err := device.WriteByteRegister(registerSysrangeIntermeasurementPeriod, byte(periodRegisterValue)); err != nil {
		return err
	}

	if err := device.WriteByteRegister(registerSysrangeStart, 0x03); err != nil {
		return err
	}

	return nil
}

// StartAmbientContinuous - Starts continuous ambient light measurements with the given period in ms
// (10 ms resolution; defaults to 500 ms if not specified).
//
// The period must be greater than the time it takes to perform a
// measurement. See section 2.4.4 ("Continuous mode limits") in the datasheet
// for details.
func (device Vl6180x) StartAmbientContinuous(period uint16) error {
	periodRegisterValue := uint16(period/10 - 1)

	if periodRegisterValue > 254 {
		periodRegisterValue = 254
	}

	if err := device.WriteByteRegister(registerSysalsIntermeasurementPeriod, byte(periodRegisterValue)); err != nil {
		return err
	}

	if err := device.WriteByteRegister(registerSysalsStart, 0x03); err != nil {
		return err
	}

	return nil
}

// StartInterleavedContinuous - Starts continuous interleaved measurements with the given period in ms
// (10 ms resolution; defaults to 500 ms if not specified). In this mode, each
// ambient light measurement is immediately followed by a range measurement.
//
// The datasheet recommends using this mode instead of running "range and ALS
// continuous modes simultaneously (i.e. asynchronously)".
//
// The period must be greater than the time it takes to perform both
// measurements. See section 2.4.4 ("Continuous mode limits") in the datasheet
// for details.
func (device Vl6180x) StartInterleavedContinuous(period uint16) error {
	periodRegisterValue := uint16(period/10 - 1)

	if periodRegisterValue > 254 {
		periodRegisterValue = 254
	}

	if err := device.WriteByteRegister(registerInterleavedModeEnable, 1); err != nil {
		return err
	}

	if err := device.WriteByteRegister(registerSysalsIntermeasurementPeriod, byte(periodRegisterValue)); err != nil {
		return err
	}

	if err := device.WriteByteRegister(registerSysalsStart, 0x03); err != nil {
		return err
	}

	return nil
}

// StopContinuous - Stops continuous mode. This will actually start a single measurement of range
// and/or ambient light if continuous mode is not active, so it's a good idea to
// wait a few hundred ms after calling this function to let that complete
// before starting continuous mode again or taking a reading.
func (device Vl6180x) StopContinuous() error {
	settings := registerSettingsTable{
		{registerSysrangeStart, 0x01},
		{registerSysalsStart, 0x01},
		{registerInterleavedModeEnable, 0},
	}

	return device.setRegisters(settings)
}

// IsRangeReadingAvailable - return true if range reading is available
func (device Vl6180x) IsRangeReadingAvailable() (bool, error) {
	if available, err := device.ReadByteRegister(registerResultInterruptStatusGpio); err == nil {
		if available&0x04 != 0 {
			return true, nil
		} else {
			return false, nil
		}
	} else {
		return true, err
	}
}

// PeekRange - check if range reading is available. If it is, read it
// The function returns three values:
//  err - not nil in case of error
//  valueAvailable - true if range reading was available, false if reading is not yet available
//  value - valid if valueAvailable is true
func (device Vl6180x) PeekRange() (valueAvailable bool, value byte, err error) {
	if valueAvailable, err = device.IsRangeReadingAvailable(); err != nil {
		return
	}

	if valueAvailable {
		if value, err = device.ReadByteRegister(registerResultRangeVal); err != nil {
			return
		}

		if err = device.WriteByteRegister(registerSystemInterruptClear, 0x01); err != nil {
			return
		}
	}

	return
}

// ReadRangeContinous - Returns a range reading when continuous mode is activated
// (readRangeSingle() also calls this function after starting a single-shot
// range measurement)
//  if timeout != 0, wait upto timeout millseconds for reading
func (device Vl6180x) ReadRangeContinous(timeout int) (byte, error) {
	var milliStart int

	if timeout != 0 {
		milliStart = time.Now().Nanosecond() / 1000000
	}

	for {
		valueAvailable, value, err := device.PeekRange()

		if err != nil {
			return 0xff, err
		}

		if valueAvailable {
			return value, nil
		}

		if timeout != 0 && time.Now().Nanosecond()/1000000-milliStart > timeout {
			return 0xff, Timeout{i2c.I2CdeviceError{Address: device.Address, Description: "ReadRange timeout"}}
		}
	}
}

// IsAmbientReadingAvailable - return true if ambient light reading is available
func (device Vl6180x) IsAmbientReadingAvailable() (bool, error) {
	if available, err := device.ReadByteRegister(registerResultInterruptStatusGpio); err == nil {
		if available&0x20 != 0 {
			return true, nil // Reading is available
		} else {
			return false, nil
		}
	} else {
		return true, err
	}
}

// PeekAmbient - check if ambient reading is available. If it is, read it
// The function returns three values:
//  err - not nil in case of error
//  valueAvailable - true if ambient reading was available, false if reading is not yet available
//  value - valid if valueAvailable is true
func (device Vl6180x) PeekAmbient() (valueAvailable bool, value uint16, err error) {
	if valueAvailable, err = device.IsAmbientReadingAvailable(); err != nil {
		return
	}

	if valueAvailable {
		if value, err = device.ReadWordRegister(registerResultAlsVal); err != nil {
			return
		}

		if err = device.WriteByteRegister(registerSystemInterruptClear, 0x02); err != nil {
			return
		}
	}

	return
}

// ReadAmbientContinous - Returns an ambient light reading when continuous mode is activated
// (readAmbientSingle() also calls this function after starting a single-shot
// ambient light measurement)
//  if timeout != 0, wait upto timeout millseconds for reading
func (device Vl6180x) ReadAmbientContinous(timeout int) (uint16, error) {
	var milliStart int

	if timeout != 0 {
		milliStart = time.Now().Nanosecond() / 1000000
	}

	for {
		valueAvailable, value, err := device.PeekAmbient()

		if err != nil {
			return 0, err
		}

		if valueAvailable {
			return value, nil
		}

		if timeout != 0 && time.Now().Nanosecond()/1000000-milliStart > timeout {
			return 0xff, Timeout{i2c.I2CdeviceError{Address: device.Address, Description: "ReadAmbient timeout"}}
		}
	}
}

func (device Vl6180x) SetGPIO1low() {
	device.WriteByteRegister(registerSystemModeGpio1, 0b00110000)
}
func (device Vl6180x) SetGPIO1high() {
	device.WriteByteRegister(registerSystemModeGpio1, 0b00000000)
}
