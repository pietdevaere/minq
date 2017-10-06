/*
Package minq is a minimal implementation of QUIC, as documented at
https://quicwg.github.io/. Minq partly implements draft-04.

*/
package minq

const (
    bitLatencySpin = 1 << 7
)

type MeasurementField uint8

/* Measurement data that will pass over the wire */
type MeasurementHeaderData struct{
    latencySpin bool
}

/* Store all (meta)data related to the measurement header field */
type MeasurementData struct {
	hdrData            MeasurementHeaderData
    maxPacketNumber    uint64
}

/* Encode the measurement header for transmission */
func (m *MeasurementHeaderData) encode() MeasurementField {
    var field MeasurementField = 0x00
    
    if m.latencySpin {
        field ^= bitLatencySpin
    }
        
    return field
}

/* Decode a received measurement header */
func (m *MeasurementField) decode() MeasurementHeaderData {
    var measurementHeaderData MeasurementHeaderData
    
    var latencSpin bool
    if (*m & bitLatencySpin) != 0 {
        latencSpin = true
    }
    
    measurementHeaderData = MeasurementHeaderData{
        latencSpin,
    }
    
    return measurementHeaderData
}

/* Create a new (empty) measurement struct */
func newMeasurementData() MeasurementData {
    return MeasurementData{
        MeasurementHeaderData{
            false,
        },
        0,
    }
}

/* Perform measurement tasks to be executed on packet reception */
func incommingMeasurementTasks(c *Connection, hdr *packetHeader){
    setOutgoingLatencySpin(c, hdr)
}

/* Look at the incomming LatencySpin, and determine what
 * the outgoing one should be */
func setOutgoingLatencySpin(c *Connection, hdr *packetHeader){
    
    /* Check if packet was received out of order. If so, ignore it */
    if hdr.PacketNumber <= c.measurement.maxPacketNumber {
        return
    } else {
        c.measurement.maxPacketNumber = hdr.PacketNumber
    }
    
    var receivedMeasurement MeasurementHeaderData
    
    receivedMeasurement = hdr.Measurement.decode()
    
    /* Server echos back the latest LatencySpinBit seen */
    if c.role == RoleServer{
        c.measurement.hdrData.latencySpin = receivedMeasurement.latencySpin
    } else {
        c.measurement.hdrData.latencySpin = !receivedMeasurement.latencySpin
    }
    
}
