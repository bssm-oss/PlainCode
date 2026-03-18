package policy

import "fmt"

// Permission represents an access level.
type Permission string

const (
	PermDeny         Permission = "deny"
	PermAsk          Permission = "ask"
	PermAllow        Permission = "allow"
	PermOwnedOnly    Permission = "owned-only"
	PermOwnedOrShared Permission = "owned-or-shared"
	PermAllowLimited Permission = "allow-limited"
	PermAllowAllowlist Permission = "allow-allowlist"
)

// Profile defines the autonomy constraints for a backend execution.
type Profile struct {
	Name           string
	Tools          Permission
	FileWrite      Permission
	Shell          Permission
	Network        Permission
	DangerousFlags bool
}

// DefaultProfiles returns the built-in approval profiles.
func DefaultProfiles() map[string]Profile {
	return map[string]Profile{
		"plan": {
			Name:      "plan",
			Tools:     PermDeny,
			FileWrite: PermDeny,
			Shell:     PermDeny,
			Network:   PermDeny,
		},
		"patch": {
			Name:      "patch",
			Tools:     PermAsk,
			FileWrite: PermOwnedOnly,
			Shell:     PermAsk,
			Network:   PermDeny,
		},
		"workspace-auto": {
			Name:      "workspace-auto",
			Tools:     PermAllow,
			FileWrite: PermOwnedOrShared,
			Shell:     PermAllowLimited,
			Network:   PermAllowAllowlist,
		},
		"sandbox-auto": {
			Name:           "sandbox-auto",
			Tools:          PermAllow,
			FileWrite:      PermAllow,
			Shell:          PermAllow,
			Network:        PermAllow,
			DangerousFlags: false,
		},
		"full-trust": {
			Name:           "full-trust",
			Tools:          PermAllow,
			FileWrite:      PermAllow,
			Shell:          PermAllow,
			Network:        PermAllow,
			DangerousFlags: true,
		},
	}
}

// Resolve returns the profile for the given name.
func Resolve(name string) (Profile, error) {
	profiles := DefaultProfiles()
	p, ok := profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown approval profile: %s", name)
	}
	return p, nil
}
