package bedrock

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// EmailSettings returns the per-assembly email delivery configuration. When no
// settings have been saved yet it returns a zero-value record with sensible
// SMTP defaults (submission port 587 with STARTTLS) and Method == EmailMethodNone.
func (db *DB) EmailSettings() (*EmailSettings, error) {
	var settings EmailSettings
	err := db.conn.Get(&settings, "SELECT * FROM email_settings ORDER BY id ASC LIMIT 1")
	if errors.Is(err, sql.ErrNoRows) {
		return &EmailSettings{
			Method:       EmailMethodNone,
			SMTPPort:     587,
			SMTPSecurity: SMTPSecuritySTARTTLS,
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get email settings: %w", err)
	}
	return &settings, nil
}

// UpdateEmailSettings writes the email delivery configuration, creating the
// single settings row if it does not yet exist. OAuth token fields are
// preserved across updates by callers that re-supply them; callers editing
// connection settings should read the current settings first so they don't
// clobber a live token. Use UpdateOAuthTokens to persist tokens in isolation.
func (db *DB) UpdateEmailSettings(s EmailSettings) (*EmailSettings, error) {
	if s.SMTPPort == 0 {
		s.SMTPPort = 587
	}

	fields := map[string]any{
		"method":              s.Method,
		"from_name":           s.FromName,
		"from_address":        s.FromAddress,
		"reply_to":            s.ReplyTo,
		"smtp_host":           s.SMTPHost,
		"smtp_port":           s.SMTPPort,
		"smtp_username":       s.SMTPUsername,
		"smtp_password":       s.SMTPPassword,
		"smtp_security":       s.SMTPSecurity,
		"oauth_client_id":     s.OAuthClientID,
		"oauth_client_secret": s.OAuthClientSecret,
		"oauth_refresh_token": s.OAuthRefreshToken,
		"oauth_access_token":  s.OAuthAccessToken,
		"oauth_token_expiry":  s.OAuthTokenExpiry,
	}

	var existingID ID
	err := db.conn.Get(&existingID, "SELECT id FROM email_settings ORDER BY id ASC LIMIT 1")
	switch {
	case errors.Is(err, sql.ErrNoRows):
		query, args := db.sq.Insert("email_settings").SetMap(fields).Suffix("RETURNING *").MustSql()
		var saved EmailSettings
		if err := db.conn.Get(&saved, query, args...); err != nil {
			return nil, fmt.Errorf("failed to insert email settings: %w", err)
		}
		return &saved, nil
	case err != nil:
		return nil, fmt.Errorf("failed to check existing email settings: %w", err)
	default:
		query, args := db.sq.Update("email_settings").SetMap(fields).Where("id = ?", existingID).Suffix("RETURNING *").MustSql()
		var saved EmailSettings
		if err := db.conn.Get(&saved, query, args...); err != nil {
			return nil, fmt.Errorf("failed to update email settings: %w", err)
		}
		return &saved, nil
	}
}

// UpdateOAuthTokens persists the OAuth access/refresh tokens and expiry on the
// existing settings row without disturbing the other fields. It returns an
// error if no settings row exists yet (configure the connection first).
func (db *DB) UpdateOAuthTokens(refreshToken, accessToken string, expiry time.Time) (*EmailSettings, error) {
	var existingID ID
	err := db.conn.Get(&existingID, "SELECT id FROM email_settings ORDER BY id ASC LIMIT 1")
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("email settings not configured; save connection settings before connecting an account")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email settings: %w", err)
	}

	fields := map[string]any{
		"oauth_access_token": accessToken,
		"oauth_token_expiry": expiry,
	}
	// A refresh token is only returned by Google on the first consent; don't
	// overwrite a stored one with an empty value on later refreshes.
	if refreshToken != "" {
		fields["oauth_refresh_token"] = refreshToken
	}

	query, args := db.sq.Update("email_settings").SetMap(fields).Where("id = ?", existingID).Suffix("RETURNING *").MustSql()
	var saved EmailSettings
	if err := db.conn.Get(&saved, query, args...); err != nil {
		return nil, fmt.Errorf("failed to update oauth tokens: %w", err)
	}
	return &saved, nil
}
