package mackerel

// Invitation information
type Invitation struct {
	Email     string `json:"email,omitempty"`
	Authority string `json:"authority,omitempty"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
}

// FindInvitations finds invitations.
func (c *Client) FindInvitations() ([]*Invitation, error) {
	data, err := requestGet[struct {
		Invitations []*Invitation `json:"invitations"`
	}](c, "/api/v0/invitations")
	if err != nil {
		return nil, err
	}
	return data.Invitations, nil
}

// CreateInvitation creates a invitation.
func (c *Client) CreateInvitation(param *Invitation) (*Invitation, error) {
	return requestPost[Invitation](c, "/api/v0/invitations", param)
}
