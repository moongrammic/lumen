cat << 'INNER_EOF' >> backend/internal/repository/guild.go

func (r *GuildRepository) GetUserGuilds(ctx context.Context, userID uuid.UUID) ([]*domain.Guild, error) {
	var guilds []*domain.Guild
	if err := r.db.WithContext(ctx).
		Joins("JOIN guild_members ON guild_members.guild_id = guilds.id").
		Where("guild_members.user_id = ?", userID).
		Find(&guilds).Error; err != nil {
		return nil, err
	}
	return guilds, nil
}
INNER_EOF
