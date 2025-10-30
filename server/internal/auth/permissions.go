package auth

type Access uint8

const (
	AccessNone    Access = 0                  // 0000
	AccessRead    Access = 1 << 0             // 0001
	AccessWrite   Access = 1<<1 | AccessRead  // 0011
	AccessExecute Access = 1<<2 | AccessWrite // 0111
	AccessAll     Access = AccessRead | AccessWrite | AccessExecute
)

type PermissionType string

const (
	PermissionAccount          PermissionType = "account"
	PermissionUser             PermissionType = "user"
	PermissionEstimate         PermissionType = "estimate"
	PermissionLineItemTemplate PermissionType = "lineItemTemplate"
)

type Permission map[PermissionType]Access

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

var UserPermissions = Permission{
	PermissionAccount:          AccessRead,
	PermissionEstimate:         AccessAll,
	PermissionLineItemTemplate: AccessAll,
	PermissionUser:             AccessRead,
}

var AdminPermissions = Permission{
	PermissionAccount:          AccessAll,
	PermissionEstimate:         AccessAll,
	PermissionLineItemTemplate: AccessAll,
	PermissionUser:             AccessAll,
}

var RoleDefinitions = map[Role]Permission{
	RoleUser:  UserPermissions,
	RoleAdmin: AdminPermissions,
}
