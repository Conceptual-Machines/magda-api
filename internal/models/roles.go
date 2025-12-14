package models

// User roles
const (
	RoleAdmin = "admin" // Unlimited credits, admin access
	RoleBeta  = "beta"  // Beta tester - generous credit allowance
	RoleUser  = "user"  // Regular user - standard credits
)

// Initial credit amounts per role
const (
	AdminInitialCredits = 999999 // Effectively unlimited
	BetaInitialCredits  = 50000  // Generous for beta testing
	UserInitialCredits  = 100    // Standard welcome bonus
)

// GetInitialCreditsForRole returns the initial credit bonus for a given role
func GetInitialCreditsForRole(role string) int {
	switch role {
	case RoleAdmin:
		return AdminInitialCredits
	case RoleBeta:
		return BetaInitialCredits
	case RoleUser:
		return UserInitialCredits
	default:
		return UserInitialCredits
	}
}

// HasUnlimitedCredits checks if a role should bypass credit deductions
func HasUnlimitedCredits(role string) bool {
	return role == RoleAdmin || role == RoleBeta
}
