// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.0
// 	protoc        v3.21.7
// source: internal/core/proto/archon.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Ship struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id          int32  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Name        string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	Ip          string `protobuf:"bytes,3,opt,name=ip,proto3" json:"ip,omitempty"`
	Port        string `protobuf:"bytes,4,opt,name=port,proto3" json:"port,omitempty"`
	PlayerCount int32  `protobuf:"varint,5,opt,name=player_count,json=playerCount,proto3" json:"player_count,omitempty"`
}

func (x *Ship) Reset() {
	*x = Ship{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_core_proto_archon_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Ship) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Ship) ProtoMessage() {}

func (x *Ship) ProtoReflect() protoreflect.Message {
	mi := &file_internal_core_proto_archon_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Ship.ProtoReflect.Descriptor instead.
func (*Ship) Descriptor() ([]byte, []int) {
	return file_internal_core_proto_archon_proto_rawDescGZIP(), []int{0}
}

func (x *Ship) GetId() int32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Ship) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Ship) GetIp() string {
	if x != nil {
		return x.Ip
	}
	return ""
}

func (x *Ship) GetPort() string {
	if x != nil {
		return x.Port
	}
	return ""
}

func (x *Ship) GetPlayerCount() int32 {
	if x != nil {
		return x.PlayerCount
	}
	return 0
}

type Account struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id               uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Username         string `protobuf:"bytes,2,opt,name=username,proto3" json:"username,omitempty"`
	Email            string `protobuf:"bytes,3,opt,name=email,proto3" json:"email,omitempty"`
	RegistrationDate string `protobuf:"bytes,4,opt,name=registration_date,json=registrationDate,proto3" json:"registration_date,omitempty"`
	Guildcard        uint64 `protobuf:"varint,5,opt,name=guildcard,proto3" json:"guildcard,omitempty"`
	Gm               bool   `protobuf:"varint,6,opt,name=gm,proto3" json:"gm,omitempty"`
	Banned           bool   `protobuf:"varint,7,opt,name=banned,proto3" json:"banned,omitempty"`
	Active           bool   `protobuf:"varint,8,opt,name=active,proto3" json:"active,omitempty"`
	TeamId           int64  `protobuf:"varint,9,opt,name=team_id,json=teamId,proto3" json:"team_id,omitempty"`
	PrivilegeLevel   []byte `protobuf:"bytes,10,opt,name=privilege_level,json=privilegeLevel,proto3" json:"privilege_level,omitempty"`
}

func (x *Account) Reset() {
	*x = Account{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_core_proto_archon_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Account) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Account) ProtoMessage() {}

func (x *Account) ProtoReflect() protoreflect.Message {
	mi := &file_internal_core_proto_archon_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Account.ProtoReflect.Descriptor instead.
func (*Account) Descriptor() ([]byte, []int) {
	return file_internal_core_proto_archon_proto_rawDescGZIP(), []int{1}
}

func (x *Account) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Account) GetUsername() string {
	if x != nil {
		return x.Username
	}
	return ""
}

func (x *Account) GetEmail() string {
	if x != nil {
		return x.Email
	}
	return ""
}

func (x *Account) GetRegistrationDate() string {
	if x != nil {
		return x.RegistrationDate
	}
	return ""
}

func (x *Account) GetGuildcard() uint64 {
	if x != nil {
		return x.Guildcard
	}
	return 0
}

func (x *Account) GetGm() bool {
	if x != nil {
		return x.Gm
	}
	return false
}

func (x *Account) GetBanned() bool {
	if x != nil {
		return x.Banned
	}
	return false
}

func (x *Account) GetActive() bool {
	if x != nil {
		return x.Active
	}
	return false
}

func (x *Account) GetTeamId() int64 {
	if x != nil {
		return x.TeamId
	}
	return 0
}

func (x *Account) GetPrivilegeLevel() []byte {
	if x != nil {
		return x.PrivilegeLevel
	}
	return nil
}

type Character struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id                uint64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Guildcard         uint64  `protobuf:"varint,2,opt,name=guildcard,proto3" json:"guildcard,omitempty"`
	GuildcardStr      []byte  `protobuf:"bytes,3,opt,name=guildcard_str,json=guildcardStr,proto3" json:"guildcard_str,omitempty"`
	Slot              uint32  `protobuf:"varint,4,opt,name=slot,proto3" json:"slot,omitempty"`
	Experience        uint32  `protobuf:"varint,5,opt,name=experience,proto3" json:"experience,omitempty"`
	Level             uint32  `protobuf:"varint,6,opt,name=level,proto3" json:"level,omitempty"`
	NameColor         uint32  `protobuf:"varint,7,opt,name=name_color,json=nameColor,proto3" json:"name_color,omitempty"`
	ModelType         int32   `protobuf:"varint,8,opt,name=model_type,json=modelType,proto3" json:"model_type,omitempty"`
	NameColorChecksum uint32  `protobuf:"varint,9,opt,name=name_color_checksum,json=nameColorChecksum,proto3" json:"name_color_checksum,omitempty"`
	SectionId         int32   `protobuf:"varint,10,opt,name=section_id,json=sectionId,proto3" json:"section_id,omitempty"`
	Class             int32   `protobuf:"varint,11,opt,name=class,proto3" json:"class,omitempty"`
	V2Flags           int32   `protobuf:"varint,12,opt,name=v2_flags,json=v2Flags,proto3" json:"v2_flags,omitempty"`
	Version           int32   `protobuf:"varint,13,opt,name=version,proto3" json:"version,omitempty"`
	V1Flags           uint32  `protobuf:"varint,14,opt,name=v1_flags,json=v1Flags,proto3" json:"v1_flags,omitempty"`
	Costume           uint32  `protobuf:"varint,15,opt,name=costume,proto3" json:"costume,omitempty"`
	Skin              uint32  `protobuf:"varint,16,opt,name=skin,proto3" json:"skin,omitempty"`
	Face              uint32  `protobuf:"varint,17,opt,name=face,proto3" json:"face,omitempty"`
	Head              uint32  `protobuf:"varint,18,opt,name=head,proto3" json:"head,omitempty"`
	Hair              uint32  `protobuf:"varint,19,opt,name=hair,proto3" json:"hair,omitempty"`
	HairRed           uint32  `protobuf:"varint,20,opt,name=hair_red,json=hairRed,proto3" json:"hair_red,omitempty"`
	HairGreen         uint32  `protobuf:"varint,21,opt,name=hair_green,json=hairGreen,proto3" json:"hair_green,omitempty"`
	HairBlue          uint32  `protobuf:"varint,22,opt,name=hair_blue,json=hairBlue,proto3" json:"hair_blue,omitempty"`
	ProportionX       float32 `protobuf:"fixed32,23,opt,name=proportion_x,json=proportionX,proto3" json:"proportion_x,omitempty"`
	ProportionY       float32 `protobuf:"fixed32,24,opt,name=proportion_y,json=proportionY,proto3" json:"proportion_y,omitempty"`
	ReadableName      string  `protobuf:"bytes,25,opt,name=readable_name,json=readableName,proto3" json:"readable_name,omitempty"`
	Name              []byte  `protobuf:"bytes,26,opt,name=name,proto3" json:"name,omitempty"`
	Playtime          uint32  `protobuf:"varint,27,opt,name=playtime,proto3" json:"playtime,omitempty"`
	Atp               uint32  `protobuf:"varint,28,opt,name=atp,proto3" json:"atp,omitempty"`
	Mst               uint32  `protobuf:"varint,29,opt,name=mst,proto3" json:"mst,omitempty"`
	Evp               uint32  `protobuf:"varint,30,opt,name=evp,proto3" json:"evp,omitempty"`
	Hp                uint32  `protobuf:"varint,31,opt,name=hp,proto3" json:"hp,omitempty"`
	Dfp               uint32  `protobuf:"varint,32,opt,name=dfp,proto3" json:"dfp,omitempty"`
	Ata               uint32  `protobuf:"varint,33,opt,name=ata,proto3" json:"ata,omitempty"`
	Lck               uint32  `protobuf:"varint,34,opt,name=lck,proto3" json:"lck,omitempty"`
	Meseta            uint32  `protobuf:"varint,35,opt,name=meseta,proto3" json:"meseta,omitempty"`
	HpMaterialsUsed   int32   `protobuf:"varint,36,opt,name=hp_materials_used,json=hpMaterialsUsed,proto3" json:"hp_materials_used,omitempty"`
	TpMaterialsUsed   int32   `protobuf:"varint,37,opt,name=tp_materials_used,json=tpMaterialsUsed,proto3" json:"tp_materials_used,omitempty"`
}

func (x *Character) Reset() {
	*x = Character{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_core_proto_archon_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Character) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Character) ProtoMessage() {}

func (x *Character) ProtoReflect() protoreflect.Message {
	mi := &file_internal_core_proto_archon_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Character.ProtoReflect.Descriptor instead.
func (*Character) Descriptor() ([]byte, []int) {
	return file_internal_core_proto_archon_proto_rawDescGZIP(), []int{2}
}

func (x *Character) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Character) GetGuildcard() uint64 {
	if x != nil {
		return x.Guildcard
	}
	return 0
}

func (x *Character) GetGuildcardStr() []byte {
	if x != nil {
		return x.GuildcardStr
	}
	return nil
}

func (x *Character) GetSlot() uint32 {
	if x != nil {
		return x.Slot
	}
	return 0
}

func (x *Character) GetExperience() uint32 {
	if x != nil {
		return x.Experience
	}
	return 0
}

func (x *Character) GetLevel() uint32 {
	if x != nil {
		return x.Level
	}
	return 0
}

func (x *Character) GetNameColor() uint32 {
	if x != nil {
		return x.NameColor
	}
	return 0
}

func (x *Character) GetModelType() int32 {
	if x != nil {
		return x.ModelType
	}
	return 0
}

func (x *Character) GetNameColorChecksum() uint32 {
	if x != nil {
		return x.NameColorChecksum
	}
	return 0
}

func (x *Character) GetSectionId() int32 {
	if x != nil {
		return x.SectionId
	}
	return 0
}

func (x *Character) GetClass() int32 {
	if x != nil {
		return x.Class
	}
	return 0
}

func (x *Character) GetV2Flags() int32 {
	if x != nil {
		return x.V2Flags
	}
	return 0
}

func (x *Character) GetVersion() int32 {
	if x != nil {
		return x.Version
	}
	return 0
}

func (x *Character) GetV1Flags() uint32 {
	if x != nil {
		return x.V1Flags
	}
	return 0
}

func (x *Character) GetCostume() uint32 {
	if x != nil {
		return x.Costume
	}
	return 0
}

func (x *Character) GetSkin() uint32 {
	if x != nil {
		return x.Skin
	}
	return 0
}

func (x *Character) GetFace() uint32 {
	if x != nil {
		return x.Face
	}
	return 0
}

func (x *Character) GetHead() uint32 {
	if x != nil {
		return x.Head
	}
	return 0
}

func (x *Character) GetHair() uint32 {
	if x != nil {
		return x.Hair
	}
	return 0
}

func (x *Character) GetHairRed() uint32 {
	if x != nil {
		return x.HairRed
	}
	return 0
}

func (x *Character) GetHairGreen() uint32 {
	if x != nil {
		return x.HairGreen
	}
	return 0
}

func (x *Character) GetHairBlue() uint32 {
	if x != nil {
		return x.HairBlue
	}
	return 0
}

func (x *Character) GetProportionX() float32 {
	if x != nil {
		return x.ProportionX
	}
	return 0
}

func (x *Character) GetProportionY() float32 {
	if x != nil {
		return x.ProportionY
	}
	return 0
}

func (x *Character) GetReadableName() string {
	if x != nil {
		return x.ReadableName
	}
	return ""
}

func (x *Character) GetName() []byte {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *Character) GetPlaytime() uint32 {
	if x != nil {
		return x.Playtime
	}
	return 0
}

func (x *Character) GetAtp() uint32 {
	if x != nil {
		return x.Atp
	}
	return 0
}

func (x *Character) GetMst() uint32 {
	if x != nil {
		return x.Mst
	}
	return 0
}

func (x *Character) GetEvp() uint32 {
	if x != nil {
		return x.Evp
	}
	return 0
}

func (x *Character) GetHp() uint32 {
	if x != nil {
		return x.Hp
	}
	return 0
}

func (x *Character) GetDfp() uint32 {
	if x != nil {
		return x.Dfp
	}
	return 0
}

func (x *Character) GetAta() uint32 {
	if x != nil {
		return x.Ata
	}
	return 0
}

func (x *Character) GetLck() uint32 {
	if x != nil {
		return x.Lck
	}
	return 0
}

func (x *Character) GetMeseta() uint32 {
	if x != nil {
		return x.Meseta
	}
	return 0
}

func (x *Character) GetHpMaterialsUsed() int32 {
	if x != nil {
		return x.HpMaterialsUsed
	}
	return 0
}

func (x *Character) GetTpMaterialsUsed() int32 {
	if x != nil {
		return x.TpMaterialsUsed
	}
	return 0
}

type GuildcardEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id              uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Guildcard       uint64 `protobuf:"varint,2,opt,name=guildcard,proto3" json:"guildcard,omitempty"`
	FriendGuildcard uint64 `protobuf:"varint,3,opt,name=friend_guildcard,json=friendGuildcard,proto3" json:"friend_guildcard,omitempty"`
	Name            []byte `protobuf:"bytes,4,opt,name=name,proto3" json:"name,omitempty"`
	TeamName        []byte `protobuf:"bytes,5,opt,name=team_name,json=teamName,proto3" json:"team_name,omitempty"`
	Description     []byte `protobuf:"bytes,6,opt,name=description,proto3" json:"description,omitempty"`
	Language        uint32 `protobuf:"varint,7,opt,name=language,proto3" json:"language,omitempty"`
	SectionId       uint32 `protobuf:"varint,8,opt,name=section_id,json=sectionId,proto3" json:"section_id,omitempty"`
	Class           uint32 `protobuf:"varint,9,opt,name=class,proto3" json:"class,omitempty"`
	Comment         []byte `protobuf:"bytes,10,opt,name=comment,proto3" json:"comment,omitempty"`
}

func (x *GuildcardEntry) Reset() {
	*x = GuildcardEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_core_proto_archon_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GuildcardEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GuildcardEntry) ProtoMessage() {}

func (x *GuildcardEntry) ProtoReflect() protoreflect.Message {
	mi := &file_internal_core_proto_archon_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GuildcardEntry.ProtoReflect.Descriptor instead.
func (*GuildcardEntry) Descriptor() ([]byte, []int) {
	return file_internal_core_proto_archon_proto_rawDescGZIP(), []int{3}
}

func (x *GuildcardEntry) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *GuildcardEntry) GetGuildcard() uint64 {
	if x != nil {
		return x.Guildcard
	}
	return 0
}

func (x *GuildcardEntry) GetFriendGuildcard() uint64 {
	if x != nil {
		return x.FriendGuildcard
	}
	return 0
}

func (x *GuildcardEntry) GetName() []byte {
	if x != nil {
		return x.Name
	}
	return nil
}

func (x *GuildcardEntry) GetTeamName() []byte {
	if x != nil {
		return x.TeamName
	}
	return nil
}

func (x *GuildcardEntry) GetDescription() []byte {
	if x != nil {
		return x.Description
	}
	return nil
}

func (x *GuildcardEntry) GetLanguage() uint32 {
	if x != nil {
		return x.Language
	}
	return 0
}

func (x *GuildcardEntry) GetSectionId() uint32 {
	if x != nil {
		return x.SectionId
	}
	return 0
}

func (x *GuildcardEntry) GetClass() uint32 {
	if x != nil {
		return x.Class
	}
	return 0
}

func (x *GuildcardEntry) GetComment() []byte {
	if x != nil {
		return x.Comment
	}
	return nil
}

type PlayerOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id        uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	KeyConfig []byte `protobuf:"bytes,2,opt,name=key_config,json=keyConfig,proto3" json:"key_config,omitempty"`
}

func (x *PlayerOptions) Reset() {
	*x = PlayerOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_internal_core_proto_archon_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *PlayerOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*PlayerOptions) ProtoMessage() {}

func (x *PlayerOptions) ProtoReflect() protoreflect.Message {
	mi := &file_internal_core_proto_archon_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use PlayerOptions.ProtoReflect.Descriptor instead.
func (*PlayerOptions) Descriptor() ([]byte, []int) {
	return file_internal_core_proto_archon_proto_rawDescGZIP(), []int{4}
}

func (x *PlayerOptions) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *PlayerOptions) GetKeyConfig() []byte {
	if x != nil {
		return x.KeyConfig
	}
	return nil
}

var File_internal_core_proto_archon_proto protoreflect.FileDescriptor

var file_internal_core_proto_archon_proto_rawDesc = []byte{
	0x0a, 0x20, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f, 0x63, 0x6f, 0x72, 0x65, 0x2f,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2f, 0x61, 0x72, 0x63, 0x68, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x12, 0x06, 0x61, 0x72, 0x63, 0x68, 0x6f, 0x6e, 0x22, 0x71, 0x0a, 0x04, 0x53, 0x68,
	0x69, 0x70, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x05, 0x52, 0x02,
	0x69, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x70, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x02, 0x69, 0x70, 0x12, 0x12, 0x0a, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x70, 0x6f, 0x72, 0x74, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x6c,
	0x61, 0x79, 0x65, 0x72, 0x5f, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x05,
	0x52, 0x0b, 0x70, 0x6c, 0x61, 0x79, 0x65, 0x72, 0x43, 0x6f, 0x75, 0x6e, 0x74, 0x22, 0x98, 0x02,
	0x0a, 0x07, 0x41, 0x63, 0x63, 0x6f, 0x75, 0x6e, 0x74, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1a, 0x0a, 0x08, 0x75, 0x73, 0x65,
	0x72, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x08, 0x75, 0x73, 0x65,
	0x72, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x18, 0x03,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65, 0x6d, 0x61, 0x69, 0x6c, 0x12, 0x2b, 0x0a, 0x11, 0x72,
	0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x64, 0x61, 0x74, 0x65,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x10, 0x72, 0x65, 0x67, 0x69, 0x73, 0x74, 0x72, 0x61,
	0x74, 0x69, 0x6f, 0x6e, 0x44, 0x61, 0x74, 0x65, 0x12, 0x1c, 0x0a, 0x09, 0x67, 0x75, 0x69, 0x6c,
	0x64, 0x63, 0x61, 0x72, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x67, 0x75, 0x69,
	0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x12, 0x0e, 0x0a, 0x02, 0x67, 0x6d, 0x18, 0x06, 0x20, 0x01,
	0x28, 0x08, 0x52, 0x02, 0x67, 0x6d, 0x12, 0x16, 0x0a, 0x06, 0x62, 0x61, 0x6e, 0x6e, 0x65, 0x64,
	0x18, 0x07, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x62, 0x61, 0x6e, 0x6e, 0x65, 0x64, 0x12, 0x16,
	0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x18, 0x08, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06,
	0x61, 0x63, 0x74, 0x69, 0x76, 0x65, 0x12, 0x17, 0x0a, 0x07, 0x74, 0x65, 0x61, 0x6d, 0x5f, 0x69,
	0x64, 0x18, 0x09, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x74, 0x65, 0x61, 0x6d, 0x49, 0x64, 0x12,
	0x27, 0x0a, 0x0f, 0x70, 0x72, 0x69, 0x76, 0x69, 0x6c, 0x65, 0x67, 0x65, 0x5f, 0x6c, 0x65, 0x76,
	0x65, 0x6c, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0e, 0x70, 0x72, 0x69, 0x76, 0x69, 0x6c,
	0x65, 0x67, 0x65, 0x4c, 0x65, 0x76, 0x65, 0x6c, 0x22, 0xe3, 0x07, 0x0a, 0x09, 0x43, 0x68, 0x61,
	0x72, 0x61, 0x63, 0x74, 0x65, 0x72, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01,
	0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1c, 0x0a, 0x09, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x63,
	0x61, 0x72, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x67, 0x75, 0x69, 0x6c, 0x64,
	0x63, 0x61, 0x72, 0x64, 0x12, 0x23, 0x0a, 0x0d, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x63, 0x61, 0x72,
	0x64, 0x5f, 0x73, 0x74, 0x72, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x0c, 0x67, 0x75, 0x69,
	0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x53, 0x74, 0x72, 0x12, 0x12, 0x0a, 0x04, 0x73, 0x6c, 0x6f,
	0x74, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x73, 0x6c, 0x6f, 0x74, 0x12, 0x1e, 0x0a,
	0x0a, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x0a, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x65, 0x6e, 0x63, 0x65, 0x12, 0x14, 0x0a,
	0x05, 0x6c, 0x65, 0x76, 0x65, 0x6c, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x6c, 0x65,
	0x76, 0x65, 0x6c, 0x12, 0x1d, 0x0a, 0x0a, 0x6e, 0x61, 0x6d, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x6f,
	0x72, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x6e, 0x61, 0x6d, 0x65, 0x43, 0x6f, 0x6c,
	0x6f, 0x72, 0x12, 0x1d, 0x0a, 0x0a, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x5f, 0x74, 0x79, 0x70, 0x65,
	0x18, 0x08, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x6d, 0x6f, 0x64, 0x65, 0x6c, 0x54, 0x79, 0x70,
	0x65, 0x12, 0x2e, 0x0a, 0x13, 0x6e, 0x61, 0x6d, 0x65, 0x5f, 0x63, 0x6f, 0x6c, 0x6f, 0x72, 0x5f,
	0x63, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75, 0x6d, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x11,
	0x6e, 0x61, 0x6d, 0x65, 0x43, 0x6f, 0x6c, 0x6f, 0x72, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x73, 0x75,
	0x6d, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18,
	0x0a, 0x20, 0x01, 0x28, 0x05, 0x52, 0x09, 0x73, 0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64,
	0x12, 0x14, 0x0a, 0x05, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x05, 0x52,
	0x05, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x12, 0x19, 0x0a, 0x08, 0x76, 0x32, 0x5f, 0x66, 0x6c, 0x61,
	0x67, 0x73, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x05, 0x52, 0x07, 0x76, 0x32, 0x46, 0x6c, 0x61, 0x67,
	0x73, 0x12, 0x18, 0x0a, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x18, 0x0d, 0x20, 0x01,
	0x28, 0x05, 0x52, 0x07, 0x76, 0x65, 0x72, 0x73, 0x69, 0x6f, 0x6e, 0x12, 0x19, 0x0a, 0x08, 0x76,
	0x31, 0x5f, 0x66, 0x6c, 0x61, 0x67, 0x73, 0x18, 0x0e, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x76,
	0x31, 0x46, 0x6c, 0x61, 0x67, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x73, 0x74, 0x75, 0x6d,
	0x65, 0x18, 0x0f, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x07, 0x63, 0x6f, 0x73, 0x74, 0x75, 0x6d, 0x65,
	0x12, 0x12, 0x0a, 0x04, 0x73, 0x6b, 0x69, 0x6e, 0x18, 0x10, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04,
	0x73, 0x6b, 0x69, 0x6e, 0x12, 0x12, 0x0a, 0x04, 0x66, 0x61, 0x63, 0x65, 0x18, 0x11, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x04, 0x66, 0x61, 0x63, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x68, 0x65, 0x61, 0x64,
	0x18, 0x12, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x68, 0x65, 0x61, 0x64, 0x12, 0x12, 0x0a, 0x04,
	0x68, 0x61, 0x69, 0x72, 0x18, 0x13, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x04, 0x68, 0x61, 0x69, 0x72,
	0x12, 0x19, 0x0a, 0x08, 0x68, 0x61, 0x69, 0x72, 0x5f, 0x72, 0x65, 0x64, 0x18, 0x14, 0x20, 0x01,
	0x28, 0x0d, 0x52, 0x07, 0x68, 0x61, 0x69, 0x72, 0x52, 0x65, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x68,
	0x61, 0x69, 0x72, 0x5f, 0x67, 0x72, 0x65, 0x65, 0x6e, 0x18, 0x15, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x09, 0x68, 0x61, 0x69, 0x72, 0x47, 0x72, 0x65, 0x65, 0x6e, 0x12, 0x1b, 0x0a, 0x09, 0x68, 0x61,
	0x69, 0x72, 0x5f, 0x62, 0x6c, 0x75, 0x65, 0x18, 0x16, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x68,
	0x61, 0x69, 0x72, 0x42, 0x6c, 0x75, 0x65, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x72, 0x6f, 0x70, 0x6f,
	0x72, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x78, 0x18, 0x17, 0x20, 0x01, 0x28, 0x02, 0x52, 0x0b, 0x70,
	0x72, 0x6f, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x58, 0x12, 0x21, 0x0a, 0x0c, 0x70, 0x72,
	0x6f, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x79, 0x18, 0x18, 0x20, 0x01, 0x28, 0x02,
	0x52, 0x0b, 0x70, 0x72, 0x6f, 0x70, 0x6f, 0x72, 0x74, 0x69, 0x6f, 0x6e, 0x59, 0x12, 0x23, 0x0a,
	0x0d, 0x72, 0x65, 0x61, 0x64, 0x61, 0x62, 0x6c, 0x65, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x19,
	0x20, 0x01, 0x28, 0x09, 0x52, 0x0c, 0x72, 0x65, 0x61, 0x64, 0x61, 0x62, 0x6c, 0x65, 0x4e, 0x61,
	0x6d, 0x65, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x1a, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1a, 0x0a, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x1b, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x08, 0x70, 0x6c, 0x61, 0x79, 0x74, 0x69,
	0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x61, 0x74, 0x70, 0x18, 0x1c, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x03, 0x61, 0x74, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x6d, 0x73, 0x74, 0x18, 0x1d, 0x20, 0x01, 0x28,
	0x0d, 0x52, 0x03, 0x6d, 0x73, 0x74, 0x12, 0x10, 0x0a, 0x03, 0x65, 0x76, 0x70, 0x18, 0x1e, 0x20,
	0x01, 0x28, 0x0d, 0x52, 0x03, 0x65, 0x76, 0x70, 0x12, 0x0e, 0x0a, 0x02, 0x68, 0x70, 0x18, 0x1f,
	0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x68, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x64, 0x66, 0x70, 0x18,
	0x20, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x64, 0x66, 0x70, 0x12, 0x10, 0x0a, 0x03, 0x61, 0x74,
	0x61, 0x18, 0x21, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x61, 0x74, 0x61, 0x12, 0x10, 0x0a, 0x03,
	0x6c, 0x63, 0x6b, 0x18, 0x22, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x03, 0x6c, 0x63, 0x6b, 0x12, 0x16,
	0x0a, 0x06, 0x6d, 0x65, 0x73, 0x65, 0x74, 0x61, 0x18, 0x23, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06,
	0x6d, 0x65, 0x73, 0x65, 0x74, 0x61, 0x12, 0x2a, 0x0a, 0x11, 0x68, 0x70, 0x5f, 0x6d, 0x61, 0x74,
	0x65, 0x72, 0x69, 0x61, 0x6c, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x64, 0x18, 0x24, 0x20, 0x01, 0x28,
	0x05, 0x52, 0x0f, 0x68, 0x70, 0x4d, 0x61, 0x74, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x73, 0x55, 0x73,
	0x65, 0x64, 0x12, 0x2a, 0x0a, 0x11, 0x74, 0x70, 0x5f, 0x6d, 0x61, 0x74, 0x65, 0x72, 0x69, 0x61,
	0x6c, 0x73, 0x5f, 0x75, 0x73, 0x65, 0x64, 0x18, 0x25, 0x20, 0x01, 0x28, 0x05, 0x52, 0x0f, 0x74,
	0x70, 0x4d, 0x61, 0x74, 0x65, 0x72, 0x69, 0x61, 0x6c, 0x73, 0x55, 0x73, 0x65, 0x64, 0x22, 0xa7,
	0x02, 0x0a, 0x0e, 0x47, 0x75, 0x69, 0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x1c, 0x0a, 0x09, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x18, 0x02,
	0x20, 0x01, 0x28, 0x04, 0x52, 0x09, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x12,
	0x29, 0x0a, 0x10, 0x66, 0x72, 0x69, 0x65, 0x6e, 0x64, 0x5f, 0x67, 0x75, 0x69, 0x6c, 0x64, 0x63,
	0x61, 0x72, 0x64, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x0f, 0x66, 0x72, 0x69, 0x65, 0x6e,
	0x64, 0x47, 0x75, 0x69, 0x6c, 0x64, 0x63, 0x61, 0x72, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61,
	0x6d, 0x65, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x1b,
	0x0a, 0x09, 0x74, 0x65, 0x61, 0x6d, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x05, 0x20, 0x01, 0x28,
	0x0c, 0x52, 0x08, 0x74, 0x65, 0x61, 0x6d, 0x4e, 0x61, 0x6d, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x64,
	0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0c,
	0x52, 0x0b, 0x64, 0x65, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x12, 0x1a, 0x0a,
	0x08, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0d, 0x52,
	0x08, 0x6c, 0x61, 0x6e, 0x67, 0x75, 0x61, 0x67, 0x65, 0x12, 0x1d, 0x0a, 0x0a, 0x73, 0x65, 0x63,
	0x74, 0x69, 0x6f, 0x6e, 0x5f, 0x69, 0x64, 0x18, 0x08, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x09, 0x73,
	0x65, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x49, 0x64, 0x12, 0x14, 0x0a, 0x05, 0x63, 0x6c, 0x61, 0x73,
	0x73, 0x18, 0x09, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x05, 0x63, 0x6c, 0x61, 0x73, 0x73, 0x12, 0x18,
	0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x18, 0x0a, 0x20, 0x01, 0x28, 0x0c, 0x52,
	0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x65, 0x6e, 0x74, 0x22, 0x3e, 0x0a, 0x0d, 0x50, 0x6c, 0x61, 0x79,
	0x65, 0x72, 0x4f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x69, 0x64, 0x12, 0x1d, 0x0a, 0x0a, 0x6b, 0x65, 0x79,
	0x5f, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0c, 0x52, 0x09, 0x6b,
	0x65, 0x79, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x42, 0x30, 0x5a, 0x2e, 0x67, 0x69, 0x74, 0x68,
	0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x64, 0x63, 0x72, 0x6f, 0x64, 0x6d, 0x61, 0x6e, 0x2f,
	0x61, 0x72, 0x63, 0x68, 0x6f, 0x6e, 0x2f, 0x69, 0x6e, 0x74, 0x65, 0x72, 0x6e, 0x61, 0x6c, 0x2f,
	0x63, 0x6f, 0x72, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_internal_core_proto_archon_proto_rawDescOnce sync.Once
	file_internal_core_proto_archon_proto_rawDescData = file_internal_core_proto_archon_proto_rawDesc
)

func file_internal_core_proto_archon_proto_rawDescGZIP() []byte {
	file_internal_core_proto_archon_proto_rawDescOnce.Do(func() {
		file_internal_core_proto_archon_proto_rawDescData = protoimpl.X.CompressGZIP(file_internal_core_proto_archon_proto_rawDescData)
	})
	return file_internal_core_proto_archon_proto_rawDescData
}

var file_internal_core_proto_archon_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_internal_core_proto_archon_proto_goTypes = []interface{}{
	(*Ship)(nil),           // 0: archon.Ship
	(*Account)(nil),        // 1: archon.Account
	(*Character)(nil),      // 2: archon.Character
	(*GuildcardEntry)(nil), // 3: archon.GuildcardEntry
	(*PlayerOptions)(nil),  // 4: archon.PlayerOptions
}
var file_internal_core_proto_archon_proto_depIdxs = []int32{
	0, // [0:0] is the sub-list for method output_type
	0, // [0:0] is the sub-list for method input_type
	0, // [0:0] is the sub-list for extension type_name
	0, // [0:0] is the sub-list for extension extendee
	0, // [0:0] is the sub-list for field type_name
}

func init() { file_internal_core_proto_archon_proto_init() }
func file_internal_core_proto_archon_proto_init() {
	if File_internal_core_proto_archon_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_internal_core_proto_archon_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Ship); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_core_proto_archon_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Account); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_core_proto_archon_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Character); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_core_proto_archon_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GuildcardEntry); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_internal_core_proto_archon_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*PlayerOptions); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_internal_core_proto_archon_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_internal_core_proto_archon_proto_goTypes,
		DependencyIndexes: file_internal_core_proto_archon_proto_depIdxs,
		MessageInfos:      file_internal_core_proto_archon_proto_msgTypes,
	}.Build()
	File_internal_core_proto_archon_proto = out.File
	file_internal_core_proto_archon_proto_rawDesc = nil
	file_internal_core_proto_archon_proto_goTypes = nil
	file_internal_core_proto_archon_proto_depIdxs = nil
}
