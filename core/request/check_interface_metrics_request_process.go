// +build !client

package request

import (
	"context"
	"fmt"
	"github.com/inexio/go-monitoringplugin"
	"github.com/inexio/thola/core/device"
	"github.com/inexio/thola/core/parser"
	"github.com/pkg/errors"
)

type interfaceCheckOutput struct {
	IfIndex       string `json:"ifIndex"`
	IfDescr       string `json:"ifDescr"`
	IfName        string `json:"ifName"`
	IfAlias       string `json:"ifAlias"`
	IfPhysAddress string `json:"ifPhysAddress"`
}

func (r *CheckInterfaceMetricsRequest) process(ctx context.Context) (Response, error) {
	r.init()

	readInterfacesResponse, err := r.getData(ctx)
	if r.mon.UpdateStatusOnError(err, monitoringplugin.UNKNOWN, "error while processing read interfaces request", true) {
		r.mon.PrintPerformanceData(false)
		return &CheckResponse{r.mon.GetInfo()}, nil
	}

	if r.PrintInterfaces {
		var interfaces []interfaceCheckOutput
		for _, interf := range readInterfacesResponse.Interfaces {
			x := interfaceCheckOutput{}
			if interf.IfIndex != nil {
				x.IfIndex = fmt.Sprint(*interf.IfIndex)
			}
			if interf.IfDescr != nil {
				x.IfDescr = *interf.IfDescr
			}
			if interf.IfName != nil {
				x.IfName = *interf.IfName
			}
			if interf.IfAlias != nil {
				x.IfAlias = *interf.IfAlias
			}
			if interf.IfPhysAddress != nil {
				x.IfPhysAddress = *interf.IfPhysAddress
			}
			interfaces = append(interfaces, x)
		}
		output, err := parser.Parse(interfaces, "json")
		if r.mon.UpdateStatusOnError(err, monitoringplugin.UNKNOWN, "error while marshalling output", true) {
			r.mon.PrintPerformanceData(false)
			return &CheckResponse{r.mon.GetInfo()}, nil
		}
		r.mon.UpdateStatus(monitoringplugin.OK, string(output))
	}

	err = addCheckInterfacePerformanceData(readInterfacesResponse.Interfaces, r.mon)
	if r.mon.UpdateStatusOnError(err, monitoringplugin.UNKNOWN, "error while adding performance data", true) {
		r.mon.PrintPerformanceData(false)
	}

	return &CheckResponse{r.mon.GetInfo()}, nil
}

func (r *CheckInterfaceMetricsRequest) getData(ctx context.Context) (*ReadInterfacesResponse, error) {
	readInterfacesRequest := ReadInterfacesRequest{ReadRequest{r.BaseRequest}}
	response, err := readInterfacesRequest.process(ctx)
	if err != nil {
		return nil, err
	}

	readInterfacesResponse := response.(*ReadInterfacesResponse)

	if len(r.Filter) > 0 {
		var interfaces []device.Interface
		for _, filter := range r.Filter {
			for _, interf := range readInterfacesResponse.Interfaces {
				if interf.IfType == nil || *interf.IfType != filter {
					interfaces = append(interfaces, interf)
				}
			}
		}
		readInterfacesResponse.Interfaces = interfaces
	}

	return readInterfacesResponse, nil
}

func addCheckInterfacePerformanceData(interfaces []device.Interface, r *monitoringplugin.Response) error {
	ifDescriptions := make(map[string]*device.Interface)

	// if the device has multiple interfaces with the same ifDescr, the ifDescr will be modified and the ifIndex will be attached
	// otherwise, the monitoring plugin will throw an error because of duplicate labels
	for i, origInterf := range interfaces {
		if origInterf.IfDescr != nil {
			if interf, ok := ifDescriptions[*origInterf.IfDescr]; ok {
				if interf != nil {
					if interf.IfIndex == nil {
						return errors.New("interface does not have an ifIndex, but ifDescr is a duplicate")
					}
					ifDescr := *interf.IfDescr + " " + fmt.Sprint(*interf.IfIndex)
					interf.IfDescr = &ifDescr
					ifDescriptions[*origInterf.IfDescr] = nil
				}
				if origInterf.IfIndex == nil {
					return errors.New("interface does not have an ifIndex, but ifDescr is a duplicate")
				}
				ifDescr := *origInterf.IfDescr + " " + fmt.Sprint(*origInterf.IfIndex)
				interfaces[i].IfDescr = &ifDescr
			} else {
				ifDescriptions[*origInterf.IfDescr] = &interfaces[i]
			}
		} else {
			if interfaces[i].IfIndex == nil {
				return errors.New("interface does not have an ifDescription and ifIndex")
			}
			x := fmt.Sprint(*interfaces[i].IfIndex)
			interfaces[i].IfDescr = &x
		}
	}

	for _, i := range interfaces {
		//error_counter_in
		if i.IfInErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_in", *i.IfInErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//error_counter_out
		if i.IfOutErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_out", *i.IfOutErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_discard_in
		if i.IfInDiscards != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_discard_in", *i.IfInDiscards, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_discard_out
		if i.IfOutDiscards != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_discard_out", *i.IfOutDiscards, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//interface_admin_status
		if i.IfAdminStatus != nil {
			value, err := i.IfAdminStatus.ToStatusCode()
			if err != nil {
				return errors.Wrap(err, "failed to convert admin status")
			}
			err = r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_admin_status", value, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//interface_oper_status
		if i.IfOperStatus != nil {
			value, err := i.IfOperStatus.ToStatusCode()
			if err != nil {
				return errors.Wrap(err, "failed to convert oper status")
			}
			err = r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_oper_status", value, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//traffic_counter_in
		if i.IfHCInOctets != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("traffic_counter_in", *i.IfHCInOctets, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfInOctets != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("traffic_counter_in", *i.IfInOctets, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//traffic_counter_out
		if i.IfHCOutOctets != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("traffic_counter_out", *i.IfHCOutOctets, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfOutOctets != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("traffic_counter_out", *i.IfOutOctets, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_unicast_in
		if i.IfHCInUcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_unicast_in", *i.IfHCInUcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfInUcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_unicast_in", *i.IfInUcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_unicast_out
		if i.IfHCOutUcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_unicast_out", *i.IfHCOutUcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfOutUcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_unicast_out", *i.IfOutUcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_multicast_in
		if i.IfHCInMulticastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_multicast_in", *i.IfHCInMulticastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfInMulticastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_multicast_in", *i.IfInMulticastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_multicast_out
		if i.IfHCOutMulticastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_multicast_out", *i.IfHCOutMulticastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfOutMulticastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_multicast_out", *i.IfOutMulticastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_broadcast_in
		if i.IfHCInBroadcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_broadcast_in", *i.IfHCInBroadcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfInBroadcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_broadcast_in", *i.IfInBroadcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//packet_counter_broadcast_out
		if i.IfHCOutBroadcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_broadcast_out", *i.IfHCOutBroadcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		} else if i.IfOutBroadcastPkts != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("packet_counter_broadcast_out", *i.IfOutBroadcastPkts, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//interface_maxspeed_in
		//interface_maxspeed_out
		if i.IfSpeed != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_maxspeed_in", *i.IfSpeed, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
			err = r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_maxspeed_out", *i.IfSpeed, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//ethernet like interface metrics
		if i.Dot3StatsAlignmentErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_alignment_errors", *i.Dot3StatsAlignmentErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsFCSErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_FCSErrors", *i.Dot3StatsFCSErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsSingleCollisionFrames != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_single_collision_frames", *i.Dot3StatsSingleCollisionFrames, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsMultipleCollisionFrames != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_multiple_collision_frames", *i.Dot3StatsMultipleCollisionFrames, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsSQETestErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_SQETest_errors", *i.Dot3StatsSQETestErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsDeferredTransmissions != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_deferred_transmissions", *i.Dot3StatsDeferredTransmissions, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsLateCollisions != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_late_collisions", *i.Dot3StatsLateCollisions, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsExcessiveCollisions != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_excessive_collisions", *i.Dot3StatsExcessiveCollisions, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsInternalMacTransmitErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_internal_mac_transmit_errors", *i.Dot3StatsInternalMacTransmitErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsCarrierSenseErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_carrier_sense_errors", *i.Dot3StatsCarrierSenseErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsFrameTooLongs != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_frame_too_longs", *i.Dot3StatsFrameTooLongs, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3StatsInternalMacReceiveErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_internal_mac_receive_errors", *i.Dot3StatsInternalMacReceiveErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.Dot3HCStatsFCSErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_dot3HCStatsFCSErrors", *i.Dot3HCStatsFCSErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.EtherStatsCRCAlignErrors != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("error_counter_CRCAlign_errors", *i.EtherStatsCRCAlignErrors, "c").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//radio interface metrics
		if i.LevelOut != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_level_out", *i.LevelOut, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.LevelIn != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_level_in", *i.LevelIn, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.MaxbitrateOut != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_maxbitrate_out", *i.MaxbitrateOut, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.MaxbitrateIn != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("interface_maxbitrate_in", *i.MaxbitrateIn, "B").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		//DWDM interface metrics
		if i.RXLevel != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("rx_level", *i.RXLevel, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}

		if i.TXLevel != nil {
			err := r.AddPerformanceDataPoint(monitoringplugin.NewPerformanceDataPoint("tx_level", *i.TXLevel, "").SetLabelTag(*i.IfDescr))
			if err != nil {
				return err
			}
		}
	}
	return nil
}
