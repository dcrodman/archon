package debug

import (
	"reflect"

	"github.com/dcrodman/archon/internal/packets"
)

// Janky (and simple) method of associating packet types with Archon's representation
// of their implementations. Of course whenever new packet types are defined they must also
// be added here in order for the sniffer to get the name correctly.

// Keeping with the janky theme, this is a cheap way to fork the packet type used
// depending on the direction the packet is coming from. True values correspond to
// client packets. false to server packets.
type multiDefinitionPacket map[bool]interface{}

var patchPacketTypes = map[uint16]interface{}{
	packets.DisconnectType: packets.PCHeader{},
	packets.RedirectType:   packets.PatchRedirect{},
	packets.PatchWelcomeType: multiDefinitionPacket{
		true:  packets.PCHeader{},
		false: packets.PatchWelcome{},
	},
	packets.PatchHandshakeType:      packets.PCHeader{},
	packets.PatchMessageType:        packets.PatchWelcomeMessage{},
	packets.PatchRedirectType:       packets.PatchRedirect{},
	packets.PatchDataAckType:        packets.PCHeader{},
	packets.PatchDirAboveType:       packets.PCHeader{},
	packets.PatchChangeDirType:      packets.ChangeDir{},
	packets.PatchCheckFileType:      packets.CheckFile{},
	packets.PatchFileListDoneType:   packets.PCHeader{},
	packets.PatchFileStatusType:     packets.FileStatus{},
	packets.PatchClientListDoneType: packets.PCHeader{},
	packets.PatchUpdateFilesType:    packets.StartFileUpdate{},
	packets.PatchFileHeaderType:     packets.FileHeader{},
	packets.PatchFileChunkType:      packets.FileChunk{},
	packets.PatchFileCompleteType:   packets.PCHeader{},
	packets.PatchUpdateCompleteType: packets.PCHeader{},
}

var packetTypes = map[uint16]interface{}{
	packets.DisconnectType:              packets.BBHeader{},
	packets.RedirectType:                packets.Redirect{},
	packets.MenuSelectType:              packets.MenuSelection{},
	packets.LoginWelcomeType:            packets.Welcome{},
	packets.LoginType:                   packets.Login{},
	packets.LoginSecurityType:           packets.Security{},
	packets.LoginClientMessageType:      packets.LoginClientMessage{},
	packets.LoginOptionsRequestType:     packets.BBHeader{},
	packets.LoginOptionsType:            packets.Options{},
	packets.LoginCharSelectType:         packets.CharacterSelection{},
	packets.LoginCharAckType:            packets.CharacterAck{},
	packets.LoginCharPreviewType:        packets.CharacterPreview{},
	packets.LoginChecksumType:           packets.BBHeader{},
	packets.LoginChecksumAckType:        packets.ChecksumAck{},
	packets.LoginGuildcardReqType:       packets.BBHeader{},
	packets.LoginGuildcardHeaderType:    packets.GuildcardHeader{},
	packets.LoginGuildcardChunkType:     packets.GuildcardChunk{},
	packets.LoginGuildcardChunkReqType:  packets.GuildcardChunkRequest{},
	packets.LoginParameterHeaderType:    packets.ParameterHeader{},
	packets.LoginParameterChunkType:     packets.ParameterChunk{},
	packets.LoginParameterChunkReqType:  packets.BBHeader{},
	packets.LoginParameterHeaderReqType: packets.BBHeader{},
	packets.LoginSetFlagType:            packets.SetFlag{},
	packets.LoginTimestampType:          packets.Timestamp{},
	packets.LoginShipListType:           packets.ShipList{},
	packets.LoginScrollMessageType:      packets.ScrollMessagePacket{},
	packets.LobbyListType:               packets.LobbyList{},
	packets.BlockListType:               packets.BlockList{},
	packets.FullCharacterType:           packets.FullCharacter{},
	packets.FullCharacterEndType:        packets.BBHeader{},
}

func getPacket(server ServerType, clientPacket bool, packetType uint16) reflect.Value {
	var (
		t     interface{}
		found bool
	)
	if server == PATCH_SERVER || server == DATA_SERVER {
		t, found = patchPacketTypes[packetType]
	} else {
		t, found = packetTypes[packetType]
	}

	if !found {
		return reflect.ValueOf(nil)
	}
	// Some packet structures may vary depending on the source, so index into the
	// definition based on which side it's coming from.
	if mvPacket, ok := t.(multiDefinitionPacket); ok {
		t = mvPacket[clientPacket]
	}

	return reflect.New(reflect.TypeOf(t))
}
