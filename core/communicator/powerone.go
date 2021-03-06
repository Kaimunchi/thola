package communicator

import (
	"context"
	"github.com/inexio/thola/core/network"
	"github.com/pkg/errors"
	"strconv"
)

type poweroneACCCommunicator struct {
	baseCommunicator
}

type poweronePCCCommunicator struct {
	baseCommunicator
}

func (c *poweroneACCCommunicator) GetUPSComponentMainsVoltageApplied(ctx context.Context) (bool, error) {
	return getPoweroneMainsVoltageApplied(ctx, ".1.3.6.1.4.1.5961.4.3.2.0")
}

func (c *poweronePCCCommunicator) GetUPSComponentMainsVoltageApplied(ctx context.Context) (bool, error) {
	return getPoweroneMainsVoltageApplied(ctx, ".1.3.6.1.4.1.5961.3.3.2.0")
}

func getPoweroneMainsVoltageApplied(ctx context.Context, oid string) (bool, error) {
	con, ok := network.DeviceConnectionFromContext(ctx)
	if !ok || con.SNMP == nil {
		return false, errors.New("no device connection available")
	}
	response, err := con.SNMP.SnmpClient.SNMPGet(ctx, oid)
	if err != nil {
		return false, errors.Wrap(err, "snmpget failed")
	}
	if len(response) != 1 {
		return false, errors.New("no or more than one snmp response available")
	}
	valueString, err := response[0].GetValueString()
	if err != nil {
		return false, errors.Wrap(err, "couldn't get string value")
	}
	value, err := strconv.Atoi(valueString)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse snmp response")
	}
	return (value & 8) == 0, nil
}
