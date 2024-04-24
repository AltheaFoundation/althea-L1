package testutil

import (
	"bytes"
	"fmt"

	"golang.org/x/exp/slices"

	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"

	ibcgotesting "github.com/AltheaFoundation/althea-L1/ibcutils/testing"
)

// RelayPacket attempts to relay the packet first on EndpointA and then on EndpointB
// if EndpointA does not contain a packet commitment for that packet. An error is returned
// if a relay step fails or the packet commitment does not exist on either endpoint.
func RelayPacket(path *ibcgotesting.Path, packet channeltypes.Packet) (*sdk.Result, error) {
	pc := path.EndpointA.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointA.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointA.Chain.App.AppCodec(), packet)) {

		// packet found, relay from A to B
		if err := path.EndpointB.UpdateClient(); err != nil {
			return nil, err
		}

		res, err := path.EndpointB.RecvPacketWithResult(packet)
		if err != nil {
			return nil, err
		}
		fmt.Printf("Acks: %v\n", path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetAllPacketAcks(path.EndpointB.Chain.GetContext()))

		ack, err := ibcgotesting.ParseAckFromEvents(res.GetEvents())
		if err != nil {
			return nil, err
		}

		if err := path.EndpointA.AcknowledgePacket(packet, ack); err != nil {
			return nil, err
		}

		return res, nil
	}

	pc = path.EndpointB.Chain.App.GetIBCKeeper().ChannelKeeper.GetPacketCommitment(path.EndpointB.Chain.GetContext(), packet.GetSourcePort(), packet.GetSourceChannel(), packet.GetSequence())
	if bytes.Equal(pc, channeltypes.CommitPacket(path.EndpointB.Chain.App.AppCodec(), packet)) {

		// packet found, relay B to A
		if err := path.EndpointA.UpdateClient(); err != nil {
			return nil, err
		}

		res, err := path.EndpointA.RecvPacketWithResult(packet)
		if err != nil {
			return nil, err
		}

		ack, err := ibcgotesting.ParseAckFromEvents(res.GetEvents())
		if err != nil {
			return nil, err
		}

		if err := path.EndpointB.AcknowledgePacket(packet, ack); err != nil {
			return nil, err
		}
		return res, nil
	}

	return nil, fmt.Errorf("packet commitment does not exist on either endpoint for provided packet")
}

func FindEvent(events []sdk.Event, name string) sdk.Event {
	index := slices.IndexFunc(events, func(e sdk.Event) bool { return e.Type == name })
	if index == -1 {
		// nolint: exhaustruct
		return sdk.Event{}
	}
	return events[index]
}

func ExtractAttributes(event sdk.Event) map[string]string {
	attrs := make(map[string]string)
	if event.Attributes == nil {
		return attrs
	}
	for _, a := range event.Attributes {
		attrs[string(a.Key)] = string(a.Value)
	}
	return attrs
}
