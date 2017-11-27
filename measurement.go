/*
Package minq is a minimal implementation of QUIC, as documented at
https://quicwg.github.io/. Minq partly implements draft-04.

*/
package minq


import (
//	"fmt"
	"time"
)

const (
	latencySpinShift = 6
	latencySpinMask = ((1 << (latencySpinShift + 1)) | (1 << latencySpinShift))
	latencySpinMod = 4
	latencyValidShift = 5
	latencyValidMask = (1 << latencyValidShift)
)

const (
	latencyRxTxDelayMax = 1 * time.Millisecond
)

type MeasurementField uint8

/* Measurement data that will pass over the wire */
type MeasurementHeaderData struct{
	latencySpin     uint8
	latencyValid    bool
}

/* Store all (meta)data related to the measurement header field */
type MeasurementData struct {
	hdrData                  MeasurementHeaderData
	maxPacketNumber          uint64
	role                     uint8
	latencyRxEdgeTime        time.Time
	lastRxLatencySpin        uint8
	generatingEdge           bool
}

/* Encode the measurement header for transmission */
func (m *MeasurementHeaderData) encode() MeasurementField {
	var field MeasurementField = 0x00

	field |= MeasurementField(m.latencySpin << latencySpinShift)

	if m.latencyValid {
		field |= MeasurementField(1 << latencyValidShift)
	}

	return field
}

/* Decode a received measurement header */
func (m MeasurementField) decode() MeasurementHeaderData {
	var measurementHeaderData MeasurementHeaderData

	latencySpin := (uint8(m) & latencySpinMask) >> latencySpinShift
	latencyValid := (uint8(m) & latencyValidMask) == latencyValidMask

	measurementHeaderData = MeasurementHeaderData{
		latencySpin,
		latencyValid,
    }

	return measurementHeaderData
}

/* Create a new (empty) measurement struct */
func newMeasurementData(role uint8) MeasurementData {
	return MeasurementData{
		MeasurementHeaderData{
			0,
			true, //TODO(piet@devae.re): or should this be false?
		},
		0,                  // maxPacketNumber
		role,               // role
		time.Now(),         // latencyRxEdgeTime
		0xff,               // lastRxLatencySpin
		false,              // generatingEdge
    }
}

/* Perform measurement tasks to be executed on packet reception */
func (m *MeasurementData) incommingMeasurementTasks(hdr *packetHeader){
	m.setOutgoingLatencySpin(hdr)
}

func (m *MeasurementData) outgoingMeasurementTasks() {
	/* We are generating an edge on the outgoing spin signal
	 * so we have to see if it can be considered "valid" */
	if m.generatingEdge {
		rxTxDelta := time.Since(m.latencyRxEdgeTime)
		if rxTxDelta > latencyRxTxDelayMax {
			m.hdrData.latencyValid = false
		} else {
			m.hdrData.latencyValid = true
		}
		m.generatingEdge = false
	}
}

/* Look at the incomming LatencySpin, and determine what
 * the outgoing one should be */
func (m *MeasurementData) setOutgoingLatencySpin(hdr *packetHeader){

	/* Check if packet was received out of order. If so, ignore it */
	if hdr.PacketNumber <= m.maxPacketNumber {
		return
	} else {
		m.maxPacketNumber = hdr.PacketNumber
	}

	var receivedMeasurement MeasurementHeaderData

	receivedMeasurement = hdr.Measurement.decode()

	/* This means we are about the generate an edge on our outgoing spinbit,
	 * so we have need to store the time this has happened, so we can later decide
	 * if the outgoing edge we create is "valid" */
	if receivedMeasurement.latencySpin != m.lastRxLatencySpin {
		m.latencyRxEdgeTime = time.Now()
		m.generatingEdge = true
	}

	/* Server echos back the latest LatencySpinBit seen */
	if m.role == RoleServer{
		m.hdrData.latencySpin = receivedMeasurement.latencySpin
	} else {
		m.hdrData.latencySpin = (receivedMeasurement.latencySpin + 1) % latencySpinMod
	}

	m.lastRxLatencySpin = receivedMeasurement.latencySpin

}
