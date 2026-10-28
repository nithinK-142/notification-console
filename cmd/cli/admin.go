package main

import (
	"errors"
	"flag"
	"fmt"
	"net/mail"
	"os"
	"strconv"
	"strings"

	"github.com/nithinK-142/notification-console/internal/models"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
	"gorm.io/gorm"
)

func runAdmin(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: cli admin <create|set-active> [flags]")
	}

	command := args[0]
	if command != "create" && command != "set-active" {
		return fmt.Errorf("unknown admin command %q; use create or set-active", command)
	}

	db, closeDB, err := openDatabase()
	if err != nil {
		return err
	}
	defer closeDB()

	switch command {
	case "create":
		return createAdmin(db, args[1:])
	case "set-active":
		return setAdminActive(db, args[1:])
	default:
		return fmt.Errorf("unknown admin command %q", command)
	}
}

func createAdmin(db *gorm.DB, args []string) error {
	flags := flag.NewFlagSet("create", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	email := flags.String("email", "", "Admin email address")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	emailValue, err := normalizeEmail(*email)
	if err != nil {
		return err
	}

	password, err := readPassword("Password: ")
	if err != nil {
		return err
	}
	confirmation, err := readPassword("Confirm password: ")
	if err != nil {
		return err
	}
	if password != confirmation {
		return errors.New("passwords do not match")
	}
	if len(password) < 15 {
		return errors.New("password must contain at least 15 characters")
	}
	if len(password) > 72 {
		return errors.New("password must not exceed 72 bytes (bcrypt limit)")
	}

	var existing models.Admin
	err = db.Where("lower(email) = ?", emailValue).First(&existing).Error
	if err == nil {
		return errors.New("admin email already exists")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check existing admin: %w", err)
	}

	hash, err := HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	admin := models.Admin{Email: emailValue, PasswordHash: hash, Active: true}
	if err := db.Create(&admin).Error; err != nil {
		return fmt.Errorf("create admin (email must be unique): %w", err)
	}
	fmt.Printf("Created active admin %s (id=%d)\n", admin.Email, admin.ID)
	return nil
}

func setAdminActive(db *gorm.DB, args []string) error {
	flags := flag.NewFlagSet("set-active", flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	email := flags.String("email", "", "Admin email address")
	active := flags.String("active", "", "Desired active state: true or false")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected positional arguments")
	}
	emailValue, err := normalizeEmail(*email)
	if err != nil {
		return err
	}
	if *active != "true" && *active != "false" {
		return errors.New("provide --active=true or --active=false")
	}
	activeValue, err := strconv.ParseBool(*active)
	if err != nil {
		return errors.New("provide --active=true or --active=false")
	}

	statusMessage := ""
	err = db.Transaction(func(tx *gorm.DB) error {
		// Serialize admin status changes so concurrent commands cannot disable
		// the last active admin at the same time.
		if err := tx.Exec("LOCK TABLE admins IN EXCLUSIVE MODE").Error; err != nil {
			return fmt.Errorf("lock admins: %w", err)
		}
		var admin models.Admin
		if err := tx.Where("lower(email) = ?", emailValue).First(&admin).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("admin not found")
			}
			return fmt.Errorf("find admin: %w", err)
		}
		if admin.Active == activeValue {
			statusMessage = fmt.Sprintf("Admin %s is already active=%t", admin.Email, admin.Active)
			return nil
		}
		if !activeValue {
			var activeCount int64
			if err := tx.Model(&models.Admin{}).Where("active = ?", true).Count(&activeCount).Error; err != nil {
				return fmt.Errorf("count active admins: %w", err)
			}
			if activeCount <= 1 {
				return errors.New("cannot disable the last active admin")
			}
		}
		if err := tx.Model(&admin).Update("active", activeValue).Error; err != nil {
			return fmt.Errorf("update admin status: %w", err)
		}
		statusMessage = fmt.Sprintf("Set admin %s active=%t", admin.Email, activeValue)
		return nil
	})
	if err != nil {
		return err
	}
	fmt.Println(statusMessage)
	return nil
}

func normalizeEmail(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || len(value) > 320 {
		return "", errors.New("provide a valid --email")
	}
	parsed, err := mail.ParseAddress(value)
	if err != nil || parsed.Address != value || strings.ContainsAny(value, " \t\r\n") {
		return "", errors.New("provide a valid --email")
	}
	return value, nil
}

func readPassword(prompt string) (string, error) {
	fd := int(os.Stdin.Fd())
	if !term.IsTerminal(fd) {
		return "", errors.New("password input requires an interactive terminal")
	}
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(fd)
	fmt.Fprintln(os.Stderr)
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}

// HashPassword creates a bcrypt hash compatible with existing admin accounts.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}
