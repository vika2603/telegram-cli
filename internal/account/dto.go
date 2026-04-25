package account

// AccountDTO is the JSON shape emitted by account commands (add / login /
// list). Zero-valued Phone and APIID are omitted via pointer + omitempty so
// a bare slot serializes without stale defaults.
type AccountDTO struct {
	Name      string  `json:"name"`
	State     string  `json:"state"`
	Phone     *string `json:"phone,omitempty"`
	APIID     *int    `json:"api_id,omitempty"`
	CreatedAt int64   `json:"created_at"`
	Default   bool    `json:"default"`
}

func (d AccountDTO) Human() string {
	star := " "
	if d.Default {
		star = "*"
	}
	return star + " " + d.Name + " [" + d.State + "]"
}

func DTOFromMeta(m Meta, isDefault bool) AccountDTO {
	var phone *string
	if m.Phone != "" {
		p := m.Phone
		phone = &p
	}
	var apiID *int
	if m.APIID != 0 {
		id := m.APIID
		apiID = &id
	}
	return AccountDTO{
		Name:      m.Name,
		State:     string(m.State),
		Phone:     phone,
		APIID:     apiID,
		CreatedAt: m.CreatedAt,
		Default:   isDefault,
	}
}
