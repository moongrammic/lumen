package domain

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	PermSendMessages uint64 = 1 << iota
	PermViewChannel
	PermManageChannels
	PermManageGuild
)

type User struct {
	gorm.Model
	ID       uuid.UUID `gorm:"type:uuid;primaryKey"`
	Username string    `gorm:"uniqueIndex;not null"`
	Email    string    `gorm:"uniqueIndex;not null"`
	Password string    `gorm:"not null"`
}

type Guild struct {
	gorm.Model
	Name       string `gorm:"not null"`
	InviteCode string `gorm:"uniqueIndex;not null"`
	OwnerID    uuid.UUID
	Owner      User
}

type GuildMember struct {
	gorm.Model
	GuildID     uint      `gorm:"index;not null"`
	UserID      uuid.UUID `gorm:"type:uuid;index;not null"`
	Role        string    `gorm:"not null;default:member"`
	Permissions uint64    `gorm:"not null;default:1"`
}

func (m *GuildMember) HasPermission(perm uint64) bool {
	if m == nil {
		return false
	}
	return (m.Permissions & perm) == perm
}

type Channel struct {
	gorm.Model
	Name    string `gorm:"not null"`
	GuildID uint
	Type    string // text, voice
}

type Message struct {
	gorm.Model
	Content     string `gorm:"not null"`
	UserID      uuid.UUID
	User        User
	ChannelID   uint
	Attachments []Attachment `gorm:"foreignKey:MessageID"`
}

type Attachment struct {
	gorm.Model
	MessageID uint   `gorm:"index;not null"`
	URL       string `gorm:"not null"`
}
