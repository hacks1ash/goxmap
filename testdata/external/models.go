// Package external simulates an external (e.g., Protobuf-generated) package
// with getter methods and different field naming conventions.
package external

// ExternalUser simulates a Protobuf-generated struct with getters.
type ExternalUser struct {
	FullName  string `json:"full_name"`
	UserEmail string `json:"user_email"`
	UserAge   int    `json:"user_age"`
}

// GetFullName is a Protobuf-style getter.
func (e *ExternalUser) GetFullName() string {
	if e != nil {
		return e.FullName
	}
	return ""
}

// GetUserEmail is a Protobuf-style getter.
func (e *ExternalUser) GetUserEmail() string {
	if e != nil {
		return e.UserEmail
	}
	return ""
}

// GetUserAge is a Protobuf-style getter.
func (e *ExternalUser) GetUserAge() int {
	if e != nil {
		return e.UserAge
	}
	return 0
}

// RemoteRecord has fields with json tags for bind_json testing.
type RemoteRecord struct {
	RemoteID   string `json:"remote_id_key"`
	RemoteData string `json:"remote_data_key"`
	Status     int    `json:"status"`
}

// ExternalRole simulates an external role type.
type ExternalRole struct {
	RoleName string `json:"role_name"`
	Level    int    `json:"level"`
}
