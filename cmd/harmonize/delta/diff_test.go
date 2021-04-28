package delta_test

import (
	"testing"

	"github.com/concourse/governance"
	"github.com/concourse/governance/cmd/harmonize/delta"
	"github.com/stretchr/testify/require"
)

var basePermissions int64
var config *governance.Config
var syncedRoles []delta.DiscordRole
var syncedMembers []delta.DiscordMember

func init() {
	var err error
	basePermissions, err = governance.TeamRoleBasePermissions.Permissions()
	if err != nil {
		panic(err)
	}

	// single config for all the tests to compare "reality" against
	config = &governance.Config{
		Teams: map[string]governance.Team{
			"banana": {
				Name:       "banana",
				RawMembers: []string{"potato"},
				Discord: governance.Discord{
					Priority: 2,
					Color:    0x123456,
				},
			},
			"admin": {
				Name:       "admin",
				RawMembers: []string{"andrew"},
				Discord: governance.Discord{
					Color:    0xbeefad,
					Priority: 99,
					AddedPermissions: governance.DiscordPermissionSet{
						"ADMINISTRATOR",
					},
				},
			},
			"all": {
				Name:            "all",
				AllContributors: true,
				Discord: governance.Discord{
					Role:     "all",
					Color:    0xabcdef,
					Priority: 1,
					Sticky:   true,
				},
			},
		},
		Contributors: map[string]governance.Person{
			"andrew": {
				Name:    "andrew",
				Discord: "andrew#123",
			},
			"potato": {
				Name:    "potato",
				Discord: "potato#456",
			},
			"onion": {
				Name:    "onion",
				Discord: "onion#789",
			},
		},
	}

	syncedRoles = []delta.DiscordRole{
		{
			ID:          "all-team-id",
			Name:        "all",
			Color:       0xabcdef,
			Permissions: basePermissions,
		},
		{
			ID:          "banana-team-id",
			Name:        "banana-team",
			Color:       0x123456,
			Permissions: basePermissions,
		},
		{
			ID:          "admin-team-id",
			Name:        "admin-team",
			Color:       0xbeefad,
			Permissions: basePermissions | 0x8,
		},
	}

	syncedMembers = []delta.DiscordMember{
		{
			ID:        "andrew-id",
			Name:      "andrew#123",
			RoleNames: []string{"admin-team", "all"},
		},
		{
			ID:        "potato-id",
			Name:      "potato#456",
			RoleNames: []string{"banana-team", "all"},
		},
	}

}

func TestSynced(t *testing.T) {
	discord := fakeDiscord{
		roles:   syncedRoles,
		members: syncedMembers,
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Empty(t, diff)
}

func TestEmptyRoleState(t *testing.T) {
	discord := fakeDiscord{
		members: []delta.DiscordMember{
			{
				ID:   "andrew-id",
				Name: "andrew#123",
			},
			{
				ID:   "potato-id",
				Name: "potato#456",
			},
		},
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Equal(t, []delta.Delta{
		delta.DeltaRoleCreate{
			RoleName:    "all",
			Color:       0xabcdef,
			Permissions: basePermissions,
		},
		delta.DeltaRoleCreate{
			RoleName:    "banana-team",
			Color:       0x123456,
			Permissions: basePermissions,
		},
		delta.DeltaRoleCreate{
			RoleName:    "admin-team",
			Color:       0xbeefad,
			Permissions: basePermissions | 0x8,
		},
		delta.DeltaUserAddRole{
			UserID:   "andrew-id",
			RoleName: "admin-team",
		},
		delta.DeltaUserAddRole{
			UserID:   "andrew-id",
			RoleName: "all",
		},
		delta.DeltaUserAddRole{
			UserID:   "potato-id",
			RoleName: "all",
		},
		delta.DeltaUserAddRole{
			UserID:   "potato-id",
			RoleName: "banana-team",
		},
	}, diff)
}

func TestRoleEdit(t *testing.T) {
	discord := fakeDiscord{
		roles: []delta.DiscordRole{
			{
				ID:          "all-team-id",
				Name:        "all",
				Color:       0xabcdef,
				Permissions: basePermissions,
			},
			{
				ID:          "banana-team-id",
				Name:        "banana-team",
				Color:       0x654321,
				Permissions: basePermissions,
			},
			{
				ID:          "admin-team-id",
				Name:        "admin-team",
				Color:       0xbeefad,
				Permissions: basePermissions,
			},
		},
		members: []delta.DiscordMember{
			{
				ID:        "andrew-id",
				Name:      "andrew#123",
				RoleNames: []string{"admin-team", "all"},
			},
			{
				ID:        "potato-id",
				Name:      "potato#456",
				RoleNames: []string{"banana-team", "all"},
			},
		},
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Equal(t, []delta.Delta{
		delta.DeltaRoleEdit{
			RoleID:      "banana-team-id",
			Color:       0x123456,
			Permissions: basePermissions,
		},
		delta.DeltaRoleEdit{
			RoleID:      "admin-team-id",
			Color:       0xbeefad,
			Permissions: basePermissions | 0x8,
		},
	}, diff)
}

func TestUserRoleAddRemove(t *testing.T) {
	discord := fakeDiscord{
		roles: syncedRoles,
		members: []delta.DiscordMember{
			{
				ID:        "andrew-id",
				Name:      "andrew#123",
				RoleNames: []string{"admin-team", "all"},
			},
			{
				ID:        "potato-id",
				Name:      "potato#456",
				RoleNames: []string{"all"},
			},
			{
				ID:        "onion-id",
				Name:      "onion#789",
				RoleNames: []string{"banana-team", "all"},
			},
		},
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Equal(t, []delta.Delta{
		delta.DeltaUserAddRole{
			UserID:   "potato-id",
			RoleName: "banana-team",
		},
		delta.DeltaUserRemoveRole{
			UserID:   "onion-id",
			RoleName: "banana-team",
		},
	}, diff)
}

func TestUserRoleIgnoreUnknown(t *testing.T) {
	roles := []delta.DiscordRole{
		{
			ID:          "unknown-role-id",
			Name:        "unknown-role",
			Color:       0xff0000,
			Permissions: basePermissions,
		},
	}
	roles = append(roles, syncedRoles...)

	discord := fakeDiscord{
		roles: roles,
		members: []delta.DiscordMember{
			{
				ID:        "andrew-id",
				Name:      "andrew#123",
				RoleNames: []string{"admin-team", "all"},
			},
			{
				ID:        "potato-id",
				Name:      "potato#456",
				RoleNames: []string{"banana-team", "all"},
			},
			{
				ID:        "onion-id",
				Name:      "onion#789",
				RoleNames: []string{"all", "unknown"},
			},
		},
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Empty(t, diff)
}

func TestUserRoleIgnoreSticky(t *testing.T) {
	discord := fakeDiscord{
		roles: syncedRoles,
		members: []delta.DiscordMember{
			{
				ID:        "andrew-id",
				Name:      "andrew#123",
				RoleNames: []string{"admin-team", "all"},
			},
			{
				ID:        "potato-id",
				Name:      "potato#456",
				RoleNames: []string{"banana-team", "all"},
			},
			{
				ID:        "onion-id",
				Name:      "onion#789",
				RoleNames: []string{"all", "unknown"},
			},
			{
				ID:        "sovereign-id",
				Name:      "sovereign#789",
				RoleNames: []string{"all"},
			},
		},
	}

	diff, err := delta.Diff(config, discord)
	require.NoError(t, err)
	require.Empty(t, diff)
}

type fakeDiscord struct {
	members []delta.DiscordMember
	roles   []delta.DiscordRole
}

func (discord fakeDiscord) Members() ([]delta.DiscordMember, error) {
	return discord.members, nil
}

func (discord fakeDiscord) Roles() ([]delta.DiscordRole, error) {
	return discord.roles, nil
}
