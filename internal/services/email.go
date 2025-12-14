package services

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"html/template"
	"time"

	"github.com/Conceptual-Machines/magda-api/internal/config"
	"github.com/Conceptual-Machines/magda-api/internal/models"
	"github.com/aws/aws-sdk-go/aws"         //nolint:staticcheck // TODO: Migrate to aws-sdk-go-v2
	"github.com/aws/aws-sdk-go/aws/session" //nolint:staticcheck
	"github.com/aws/aws-sdk-go/service/ses" //nolint:staticcheck
	"gorm.io/gorm"
)

type EmailService struct {
	db        *gorm.DB
	cfg       *config.Config
	sesClient *ses.SES
}

func NewEmailService(db *gorm.DB, cfg *config.Config) *EmailService {
	// Initialize AWS SES client
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWSRegion),
	}))

	return &EmailService{
		db:        db,
		cfg:       cfg,
		sesClient: ses.New(sess),
	}
}

const (
	tokenBytes              = 32
	verificationTokenExpiry = 24 * time.Hour
)

// GenerateVerificationToken creates a new email verification token
func (s *EmailService) GenerateVerificationToken(userID uint) (string, error) {
	// Generate random token
	randomBytes := make([]byte, tokenBytes)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(randomBytes)

	// Create verification token record
	verificationToken := models.EmailVerificationToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(verificationTokenExpiry),
	}

	if err := s.db.Create(&verificationToken).Error; err != nil {
		return "", err
	}

	return token, nil
}

// SendVerificationEmail sends a verification email to the user
func (s *EmailService) SendVerificationEmail(user *models.User, token string) error {
	verificationURL := fmt.Sprintf("%s/verify-email?token=%s", s.cfg.FrontendURL, token)

	// Email template
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Verify Your Email - Musical AIDEAS</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">🎵 Musical AIDEAS</h1>
    </div>
    <div style="background-color: white; padding: 40px; border-radius: 0 0 10px 10px;
                box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
        <h2 style="color: #333;">Welcome, {{.Name}}!</h2>
        <p style="color: #666; line-height: 1.6;">
            Thank you for signing up for Musical AIDEAS beta. To complete your registration
            and start creating AI-powered music, please verify your email address.
        </p>
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.VerificationURL}}"
               style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                      color: white; padding: 15px 40px; text-decoration: none;
                      border-radius: 5px; font-weight: bold; display: inline-block;">
                Verify Email Address
            </a>
        </div>
        <p style="color: #999; font-size: 12px; margin-top: 30px;">
            If the button doesn't work, copy and paste this link into your browser:<br>
            <a href="{{.VerificationURL}}" style="color: #667eea;">{{.VerificationURL}}</a>
        </p>
        <p style="color: #999; font-size: 12px;">
            This link will expire in 24 hours. If you didn't sign up for Musical AIDEAS,
            you can safely ignore this email.
        </p>
    </div>
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>© 2025 Musical AIDEAS. All rights reserved.</p>
    </div>
</body>
</html>`

	// Parse template
	tmpl, err := template.New("verification").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	var htmlBody bytes.Buffer
	err = tmpl.Execute(&htmlBody, map[string]string{
		"Name":            user.Name,
		"VerificationURL": verificationURL,
	})
	if err != nil {
		return err
	}

	// Plain text version
	textBody := fmt.Sprintf(`Welcome to Musical AIDEAS!

Thank you for signing up, %s. To complete your registration, please verify your email address by clicking the link below:

%s

This link will expire in 24 hours.

If you didn't sign up for Musical AIDEAS, you can safely ignore this email.

---
Musical AIDEAS
AI-powered music generation for your DAW
`, user.Name, verificationURL)

	// Send email using AWS SES
	input := &ses.SendEmailInput{
		Source: aws.String(s.cfg.EmailFrom), // e.g., "Musical AIDEAS <noreply@musicalaideas.com>"
		Destination: &ses.Destination{
			ToAddresses: []*string{aws.String(user.Email)},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data:    aws.String("Verify Your Email - Musical AIDEAS"),
				Charset: aws.String("UTF-8"),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Data:    aws.String(htmlBody.String()),
					Charset: aws.String("UTF-8"),
				},
				Text: &ses.Content{
					Data:    aws.String(textBody),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err = s.sesClient.SendEmail(input)
	return err
}

// VerifyEmail verifies an email using the provided token
func (s *EmailService) VerifyEmail(token string) error {
	var verificationToken models.EmailVerificationToken
	if err := s.db.Where("token = ?", token).First(&verificationToken).Error; err != nil {
		return fmt.Errorf("invalid verification token")
	}

	// Check if already used
	if verificationToken.UsedAt != nil {
		return fmt.Errorf("verification token already used")
	}

	// Check if expired
	if time.Now().After(verificationToken.ExpiresAt) {
		return fmt.Errorf("verification token expired")
	}

	// Start transaction
	tx := s.db.Begin()

	// Mark token as used
	now := time.Now()
	verificationToken.UsedAt = &now
	if err := tx.Save(&verificationToken).Error; err != nil {
		tx.Rollback()
		return err
	}

	// Mark user as verified
	if err := tx.Model(&models.User{}).Where("id = ?", verificationToken.UserID).Updates(map[string]interface{}{
		"email_verified": true,
		"verified_at":    now,
	}).Error; err != nil {
		tx.Rollback()
		return err
	}

	tx.Commit()
	return nil
}

// ResendVerificationEmail generates a new token and resends the verification email
func (s *EmailService) ResendVerificationEmail(email string) error {
	var user models.User
	if err := s.db.Where("email = ?", email).First(&user).Error; err != nil {
		return fmt.Errorf("user not found")
	}

	if user.EmailVerified {
		return fmt.Errorf("email already verified")
	}

	// Invalidate old tokens
	s.db.Where("user_id = ? AND used_at IS NULL", user.ID).
		Update("used_at", time.Now())

	// Generate new token
	token, err := s.GenerateVerificationToken(user.ID)
	if err != nil {
		return err
	}

	// Send email
	return s.SendVerificationEmail(&user, token)
}

// SendInvitationEmail sends an invitation code to a potential user
func (s *EmailService) SendInvitationEmail(email, code, note string) error {
	signupURL := fmt.Sprintf("%s/auth/accept-invitation?email=%s&code=%s", s.cfg.FrontendURL, email, code)

	// Email template
	htmlTemplate := `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>You're Invited to Musical AIDEAS Beta!</title>
</head>
<body style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto; padding: 20px;">
    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                padding: 30px; border-radius: 10px 10px 0 0; text-align: center;">
        <h1 style="color: white; margin: 0;">🎵 Musical AIDEAS</h1>
    </div>
    <div style="background-color: white; padding: 40px; border-radius: 0 0 10px 10px;
                box-shadow: 0 2px 10px rgba(0,0,0,0.1);">
        <h2 style="color: #333;">You're Invited!</h2>
        <p style="color: #666; line-height: 1.6;">
            You've been invited to join the Musical AIDEAS beta - an AI-powered music generation
            plugin that works directly in your DAW. Click the button below to set your password and get started!
        </p>
        {{if .Note}}
        <p style="color: #666; line-height: 1.6; font-style: italic;">
            Note: {{.Note}}
        </p>
        {{end}}
        <div style="text-align: center; margin: 30px 0;">
            <a href="{{.SignupURL}}"
               style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
                      color: white; padding: 15px 40px; text-decoration: none;
                      border-radius: 5px; font-weight: bold; display: inline-block;">
                Accept Invitation & Set Password
            </a>
        </div>
        <p style="color: #999; font-size: 12px; margin-top: 30px;">
            If the button doesn't work, copy and paste this link into your browser:<br>
            <a href="{{.SignupURL}}" style="color: #667eea;">{{.SignupURL}}</a>
        </p>
    </div>
    <div style="text-align: center; padding: 20px; color: #999; font-size: 12px;">
        <p>© 2025 Musical AIDEAS. All rights reserved.</p>
    </div>
</body>
</html>`

	// Parse template
	tmpl, err := template.New("invitation").Parse(htmlTemplate)
	if err != nil {
		return err
	}

	var htmlBody bytes.Buffer
	err = tmpl.Execute(&htmlBody, map[string]string{
		"Code":      code,
		"Note":      note,
		"SignupURL": signupURL,
	})
	if err != nil {
		return err
	}

	// Plain text version
	textBody := fmt.Sprintf(`You're Invited to Musical AIDEAS Beta!

You've been invited to join the Musical AIDEAS beta - an AI-powered music generation plugin for your DAW.

Click the link below to accept your invitation and set your password:

%s

---
Musical AIDEAS
AI-powered music generation for your DAW
`, signupURL)

	// Send email using AWS SES
	input := &ses.SendEmailInput{
		Source: aws.String(s.cfg.EmailFrom),
		Destination: &ses.Destination{
			ToAddresses: []*string{aws.String(email)},
		},
		Message: &ses.Message{
			Subject: &ses.Content{
				Data:    aws.String("You're Invited to Musical AIDEAS Beta! 🎵"),
				Charset: aws.String("UTF-8"),
			},
			Body: &ses.Body{
				Html: &ses.Content{
					Data:    aws.String(htmlBody.String()),
					Charset: aws.String("UTF-8"),
				},
				Text: &ses.Content{
					Data:    aws.String(textBody),
					Charset: aws.String("UTF-8"),
				},
			},
		},
	}

	_, err = s.sesClient.SendEmail(input)
	return err
}
