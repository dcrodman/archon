package debug

import (
	"reflect"

	"github.com/dcrodman/archon/internal/commands"
)

// Janky (and simple) method of associating command types with Archon's representation
// of their implementations. Of course whenever new command types are defined they must also
// be added here in order for the sniffer to get the name correctly.

// Keeping with the janky theme, this is a cheap way to fork the command type used
// depending on the direction the command is coming from. True values correspond to
// client commands. false to server commands.
type multiDefinitionCommand map[bool]interface{}

var patchcommandTypes = map[uint16]interface{}{
	commands.DisconnectType: commands.PCHeader{},
	commands.RedirectType:   commands.PatchRedirect{},
	commands.PatchWelcomeType: multiDefinitionCommand{
		true:  commands.PCHeader{},
		false: commands.PatchWelcome{},
	},
	commands.PatchHandshakeType:      commands.PCHeader{},
	commands.PatchMessageType:        commands.PatchWelcomeMessage{},
	commands.PatchRedirectType:       commands.PatchRedirect{},
	commands.PatchDataAckType:        commands.PCHeader{},
	commands.PatchDirAboveType:       commands.PCHeader{},
	commands.PatchChangeDirType:      commands.ChangeDir{},
	commands.PatchCheckFileType:      commands.PatchCheckFile{},
	commands.PatchFileListDoneType:   commands.PCHeader{},
	commands.PatchFileStatusType:     commands.PatchFileStatus{},
	commands.PatchClientListDoneType: commands.PCHeader{},
	commands.PatchStartFileUpdateType:    commands.PatchStartFileUpdate{},
	commands.PatchFileHeaderType:     commands.PatchFileHeader{},
	commands.PatchFileChunkType:      commands.PatchFileChunk{},
	commands.PatchFileCompleteType:   commands.PCHeader{},
	commands.PatchUpdateCompleteType: commands.PCHeader{},
}

var commandTypes = map[uint16]interface{}{
	commands.DisconnectType:              commands.BBHeader{},
	commands.RedirectType:                commands.Redirect{},
	commands.MenuSelectionType:           commands.MenuSelection{},
	commands.LoginWelcomeType:            commands.Welcome{},
	commands.LoginType:                   commands.Login{},
	commands.LoginSecurityType:           commands.Security{},
	commands.LoginClientMessageType:      commands.LoginClientMessage{},
	commands.LoginOptionsRequestType:     commands.BBHeader{},
	commands.LoginOptionsType:            commands.Options{},
	commands.LoginCharSelectType:         commands.CharacterSelection{},
	commands.LoginCharAckType:            commands.CharacterAck{},
	commands.LoginCharPreviewType:        commands.CharacterPreview{},
	commands.LoginChecksumType:           commands.BBHeader{},
	commands.LoginChecksumAckType:        commands.ChecksumAck{},
	commands.LoginGuildcardReqType:       commands.BBHeader{},
	commands.LoginGuildcardHeaderType:    commands.GuildcardHeader{},
	commands.LoginGuildcardChunkType:     commands.GuildcardChunk{},
	commands.LoginGuildcardChunkReqType:  commands.GuildcardChunkRequest{},
	commands.LoginParameterHeaderType:    commands.ParameterHeader{},
	commands.LoginParameterChunkType:     commands.ParameterChunk{},
	commands.LoginParameterChunkReqType:  commands.BBHeader{},
	commands.LoginParameterHeaderReqType: commands.BBHeader{},
	commands.LoginSetFlagType:            commands.SetFlag{},
	commands.LoginTimestampType:          commands.Timestamp{},
	commands.LoginShipListType:           commands.ShipList{},
	commands.LoginScrollMessageType:      commands.ScrollMessage{},
	commands.LobbyMenuType:               commands.LobbyList{},
	commands.BlockListType:               commands.ShipMenu{},
	commands.SyncCharacterType:           commands.SyncCharacter{},
}

func getCommand(server string, clientcommand bool, commandType uint16) reflect.Value {
	var (
		t     interface{}
		found bool
	)
	if server == "PATCH:AUTH" || server == "PATCH:DATA" {
		t, found = patchcommandTypes[commandType]
	} else {
		t, found = commandTypes[commandType]
	}

	if !found {
		return reflect.ValueOf(nil)
	}
	// Some command structures may vary depending on the source, so index into the
	// definition based on which side it's coming from.
	if mvcommand, ok := t.(multiDefinitionCommand); ok {
		t = mvcommand[clientcommand]
	}

	return reflect.New(reflect.TypeOf(t))
}
