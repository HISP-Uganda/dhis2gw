package auth

type Permission struct {
	Module string
	Perms  string
}

type OrgUnit struct {
	ID    int64
	UID   string
	Code  string
	Name  string
	Path  string
	Level int
}

type User struct {
	ID       int64
	Username string
	Role     string
	Perms    map[string]string
	OrgUnits []OrgUnit
}
