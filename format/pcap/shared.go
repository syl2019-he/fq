package pcap

import (
	"github.com/wader/fq/format"
	"github.com/wader/fq/format/inet/flowsdecoder"
	"github.com/wader/fq/pkg/bitio"
	"github.com/wader/fq/pkg/decode"
)

var linkToDecodeFn = map[int]func(fd *flowsdecoder.Decoder, bs []byte) error{
	format.LinkTypeNULL:       (*flowsdecoder.Decoder).LoopbackFrame,
	format.LinkTypeETHERNET:   (*flowsdecoder.Decoder).EthernetFrame,
	format.LinkTypeLINUX_SLL:  (*flowsdecoder.Decoder).SLLPacket,
	format.LinkTypeLINUX_SLL2: (*flowsdecoder.Decoder).SLL2Packet,
}

// TODO: make some of this shared if more packet capture formats are added
func fieldFlows(d *decode.D, fd *flowsdecoder.Decoder, tcpStreamFormat decode.Group, ipv4PacketFormat decode.Group) {
	d.FieldArray("ipv4_reassembled", func(d *decode.D) {
		for _, p := range fd.IPV4Reassembled {
			br := bitio.NewBitReader(p.Datagram, -1)
			if dv, _, _ := d.TryFieldFormatBitBuf(
				"ipv4_packet",
				br,
				ipv4PacketFormat,
				nil,
			); dv == nil {
				d.FieldRootBitBuf("ipv4_packet", br)
			}
		}
	})

	d.FieldArray("tcp_connections", func(d *decode.D) {
		for _, s := range fd.TCPConnections {
			d.FieldStruct("tcp_connection", func(d *decode.D) {
				f := func(d *decode.D, td *flowsdecoder.TCPDirection, tsi format.TCPStreamIn) {
					d.FieldValueStr("ip", td.Endpoint.IP.String())
					d.FieldValueU("port", uint64(td.Endpoint.Port), format.TCPPortMap)
					d.FieldValueBool("has_start", td.HasStart)
					d.FieldValueBool("has_end", td.HasEnd)
					d.FieldValueU("skipped_bytes", td.SkippedBytes)

					br := bitio.NewBitReader(td.Buffer.Bytes(), -1)
					if dv, _, _ := d.TryFieldFormatBitBuf(
						"stream",
						br,
						tcpStreamFormat,
						tsi,
					); dv == nil {
						d.FieldRootBitBuf("stream", br)
					}
				}

				d.FieldStruct("client", func(d *decode.D) {
					f(d, &s.Client, format.TCPStreamIn{
						IsClient:        true,
						HasStart:        s.Client.HasStart,
						HasEnd:          s.Client.HasEnd,
						SkippedBytes:    s.Client.SkippedBytes,
						SourcePort:      s.Client.Endpoint.Port,
						DestinationPort: s.Server.Endpoint.Port,
					})
				})
				d.FieldStruct("server", func(d *decode.D) {
					f(d, &s.Server, format.TCPStreamIn{
						IsClient:        false,
						HasStart:        s.Server.HasStart,
						HasEnd:          s.Server.HasEnd,
						SkippedBytes:    s.Server.SkippedBytes,
						SourcePort:      s.Server.Endpoint.Port,
						DestinationPort: s.Client.Endpoint.Port,
					})
				})
			})
		}
	})
}
