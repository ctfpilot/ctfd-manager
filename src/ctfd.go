package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"strings"

	ctfd "github.com/ctfer-io/go-ctfd/api"
)

const CTFDACCESSCONFIGMAP = "ctfd-access-token"
const CTFDCHALLENGESCONFIGMAP = "ctfd-challenges"
const CTFDPAGESCONFIGMAP = "ctfd-pages"

type CTFdSetupParamsInputFile struct {
	Name    string `json:"name"`
	Content string `json:"content"` // Base64 encoded
}

type CTFdSetupParams struct {
	// CTF base info
	CTFName        string `json:"ctf_name"`
	CTFDescription string `json:"ctf_description"`
	Start          string `json:"start"` // unix timestamp
	End            string `json:"end"`   // unix timestamp

	// CTF settings
	UserMode               string        `json:"user_mode"`               // "teams" or "users"
	ChallengeVisibility    string        `json:"challenge_visibility"`    // "Public", "Private", "Admins Only"
	AccountVisibility      string        `json:"account_visibility"`      // "Public", "Private", "Admins Only"
	ScoreVisibility        string        `json:"score_visibility"`        // "Public", "Private", "Hidden", "Admins Only"
	RegistrationVisibility string        `json:"registration_visibility"` // "Public", "Private", "MajorLeagueCyber Only"
	VerifyEmails           bool          `json:"verify_emails"`
	TeamSize               int           `json:"team_size,omitempty"`
	Brackets               []CTFdBracket `json:"brackets,omitempty"` // List of brackets, optional

	// Theme
	CTFLogo      CTFdSetupParamsInputFile `json:"ctf_logo,omitempty"`
	CTFBanner    CTFdSetupParamsInputFile `json:"ctf_banner,omitempty"`
	CTFSmallIcon CTFdSetupParamsInputFile `json:"ctf_smallicon,omitempty"`
	CTFTheme     string                   `json:"ctf_theme"`
	ThemeColor   string                   `json:"theme_color,omitempty"` // Hex color code

	// Admin user
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`

	// Mail
	MailServer   string `json:"mail_server,omitempty"`
	MailPort     int    `json:"mail_port,omitempty"`
	MailUsername string `json:"mail_username,omitempty"`
	MailPassword string `json:"mail_password,omitempty"`
	MailSSL      bool   `json:"mail_ssl,omitempty"`
	MailTLS      bool   `json:"mail_tls,omitempty"`
	MailFrom     string `json:"mail_from,omitempty"` // Sender email address

	// Registration code
	RegistrationCode string `json:"registration_code,omitempty"`
}

type CTFdBracket struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"` // empty, "users" or "teams"
}

func validateCTFdSetupParams(params *CTFdSetupParams) error {
	requiredFields := "CTFName,CTFDescription,UserMode,ChallengeVisibility,AccountVisibility,ScoreVisibility,RegistrationVisibility,VerifyEmails,CTFTheme,Name,Email,Password"

	// Split the required fields into a slice
	requiredFieldsSlice := strings.Split(requiredFields, ",")

	// Use reflection to dynamically access struct fields
	paramsValue := reflect.ValueOf(params).Elem()
	for _, field := range requiredFieldsSlice {
		// Check if the field exists in the struct
		if fieldValue := paramsValue.FieldByName(field); !fieldValue.IsValid() {
			return errors.New("invalid field: " + field)
		}

		// Check if the field is a string and trim it
		fieldValue := paramsValue.FieldByName(field)
		if !fieldValue.IsValid() {
			return errors.New("invalid field: " + field)
		}
		if fieldValue.Kind() == reflect.String {
			trimmedValue := strings.TrimSpace(fieldValue.String())
			fieldValue.SetString(trimmedValue)
			if len(trimmedValue) == 0 {
				return errors.New("missing required field: " + field)
			}
		}

		// Validate select inputs
		validValues := []string{}
		selectCheck := true
		switch field {
		case "UserMode":
			validValues = []string{"users", "teams"}
		case "ChallengeVisibility":
			validValues = []string{"public", "private", "admins"}
		case "AccountVisibility":
			validValues = []string{"public", "private", "admins"}
		case "ScoreVisibility":
			validValues = []string{"public", "private", "hidden", "admins"}
		case "RegistrationVisibility":
			validValues = []string{"public", "private", "Mmlc"}
		default:
			// No specific validation for other fields
			selectCheck = false
			continue
		}

		// Validate user mode
		if selectCheck {
			if !slices.Contains(validValues, fieldValue.String()) {
				return errors.New("invalid value for " + field + ": " + fieldValue.String() + ", valid values are: " + strings.Join(validValues, ", "))
			}
		}
	}

	// Validate start and end timestamps, if set
	if params.Start != "" && params.End != "" {
		start, err := strconv.ParseInt(params.Start, 10, 64)
		if err != nil {
			return errors.New("invalid start timestamp: " + err.Error())
		}
		if start < 0 {
			return errors.New("start timestamp must be a positive integer")
		}

		end, err := strconv.ParseInt(params.End, 10, 64)
		if err != nil {
			return errors.New("invalid end timestamp: " + err.Error())
		}
		if start >= end {
			return errors.New("start timestamp must be less than end timestamp")
		}
	}

	// Validate team size, if set
	if params.TeamSize != 0 && params.TeamSize < 1 {
		// Team size must be greater than 0
		return errors.New("team size must be greater than 0")
	}

	// Validate mail port, if set
	if params.MailPort != 0 {
		if params.MailPort < 1 || params.MailPort > 65535 {
			return errors.New("mail port must be between 1 and 65535")
		}
	}

	// Validate brackets, if set
	if len(params.Brackets) > 0 {
		for _, bracket := range params.Brackets {
			if bracket.Name == "" {
				return errors.New("bracket name cannot be empty")
			}
			bracket.Name = strings.TrimSpace(bracket.Name)
			if bracket.Type != "" && bracket.Type != "users" && bracket.Type != "teams" {
				return errors.New("invalid bracket type: " + bracket.Type + ", valid values are: \"\", \"users\", \"teams\"")
			}
			if bracket.Description != "" && len(bracket.Description) > 255 {
				return errors.New("bracket description cannot exceed 255 characters")
			}
		}
	}

	return nil
}

func getCTFdInputFile(data CTFdSetupParamsInputFile) *ctfd.InputFile {
	// Check if the field is empty
	if data.Content == "" || data.Name == "" {
		return nil
	}

	// Decode the base64 string
	decodedData, err := base64.StdEncoding.DecodeString(data.Content)
	if err != nil {
		log.Println("Error decoding base64 string:", err)
		return nil
	}

	// Check if the decoded data is empty
	if len(decodedData) == 0 {
		log.Println("Decoded data is empty")
		return nil
	}

	// Create a new InputFile with the decoded data
	return &ctfd.InputFile{
		Name:    data.Name,
		Content: decodedData,
	}
}

func convertCTFdSetupParamsToSetupParams(params *CTFdSetupParams) *ctfd.SetupParams {
	logoInputFile := getCTFdInputFile(params.CTFLogo)
	bannerInputFile := getCTFdInputFile(params.CTFBanner)
	smallIconInputFile := getCTFdInputFile(params.CTFSmallIcon)

	// Map the CTFdSetupParams to CTFd SetupParams
	return &ctfd.SetupParams{
		CTFName:                params.CTFName,
		CTFDescription:         params.CTFDescription,
		Start:                  params.Start,
		End:                    params.End,
		UserMode:               params.UserMode,
		ChallengeVisibility:    params.ChallengeVisibility,
		AccountVisibility:      params.AccountVisibility,
		ScoreVisibility:        params.ScoreVisibility,
		RegistrationVisibility: params.RegistrationVisibility,
		VerifyEmails:           params.VerifyEmails,
		TeamSize:               &params.TeamSize,
		CTFLogo:                logoInputFile,
		CTFBanner:              bannerInputFile,
		CTFSmallIcon:           smallIconInputFile,
		CTFTheme:               params.CTFTheme,
		ThemeColor:             params.ThemeColor,
		Name:                   params.Name,
		Email:                  params.Email,
		Password:               params.Password,
	}
}

func setupCTFd(client *ctfd.Client, params *CTFdSetupParams) error {
	log.Println("Setting up CTFd")

	// Validate the parameters
	if err := validateCTFdSetupParams(params); err != nil {
		return err
	}

	// Convert to CTFd SetupParams
	setupParams := convertCTFdSetupParamsToSetupParams(params)

	// Set up CTFd
	if err := client.Setup(setupParams); err != nil {
		log.Println("Error setting up CTFd:", err)
		return errors.New("error setting up CTFd: " + err.Error())
	}

	log.Println("CTFd setup completed successfully")

	return nil
}

func setupCTFdBrackets(client *ctfd.Client, params *CTFdSetupParams) error {
	log.Println("Setting up CTFd brackets")

	// Validate the parameters
	if err := validateCTFdSetupParams(params); err != nil {
		return err
	}

	// Check if brackets are provided
	if len(params.Brackets) == 0 {
		log.Println("No brackets to set up")
		return nil
	}

	// Set up CTFd brackets
	for _, bracket := range params.Brackets {
		if returnedBracket, err := client.PostBrackets(&ctfd.PostBracketsParams{
			ID:          0,
			Name:        bracket.Name,
			Description: bracket.Description,
			Type:        bracket.Type,
		}); err != nil {
			return errors.New("error setting up CTFd bracket '" + bracket.Name + "': " + err.Error())
		} else {
			log.Printf("CTFd bracket '%s' set up successfully with ID %d", returnedBracket.Name, int(returnedBracket.ID))
		}
	}

	log.Println("CTFd brackets setup completed successfully")

	return nil
}

func setupCTFdMailSettings(client *ctfd.Client, params *CTFdSetupParams) error {
	log.Println("Setting up CTFd mail settings")

	// Validate the parameters
	if err := validateCTFdSetupParams(params); err != nil {
		return err
	}

	userAuth := true

	// Set up CTFd mail settings
	if err := client.PatchConfigs(&ctfd.PatchConfigsParams{
		MailServer:   &params.MailServer,
		MailPort:     func() *string { port := strconv.Itoa(params.MailPort); return &port }(),
		MailUseAuth:  &userAuth,
		MailUsername: &params.MailUsername,
		MailPassword: &params.MailPassword,
		MailFromAddr: &params.MailFrom,
		MailSSL:      &params.MailSSL,
		MailTLS:      &params.MailTLS,
	}); err != nil {
		return errors.New("error setting up CTFd mail settings: " + err.Error())
	}

	log.Println("CTFd mail settings setup completed successfully")

	return nil
}

func setupCTFdRegistrationCode(client *ctfd.Client, params *CTFdSetupParams) error {
	log.Println("Setting up CTFd registration code")

	// Validate the parameters
	if err := validateCTFdSetupParams(params); err != nil {
		return err
	}

	// Set up CTFd registration code
	if err := client.PatchConfigs(&ctfd.PatchConfigsParams{
		RegistrationCode: &params.RegistrationCode,
	}); err != nil {
		return errors.New("error setting up CTFd registration code: " + err.Error())
	}

	log.Println("CTFd registration code setup completed successfully")

	return nil
}

func setupCTFdAccessToken(client *ctfd.Client, params *CTFdSetupParams) error {
	log.Println("Setting up CTFd access token")

	// Validate the parameters
	if err := validateCTFdSetupParams(params); err != nil {
		return err
	}

	// Set up CTFd access token
	token, err := client.PostTokens(&ctfd.PostTokensParams{
		Expiration:  "2222-02-02",
		Description: "Auto generated access token for CTFd manager",
	})
	if err != nil {
		return errors.New("error creating CTFd access token: " + err.Error())
	}

	// Store token in configmap
	accessToken := token.Value
	if accessToken == nil {
		return errors.New("error creating CTFd access token: token is nil")
	}

	if err := updateConfigMap(getNamespace(), CTFDACCESSCONFIGMAP, map[string]string{
		"access_token": *accessToken,
	}); err != nil {
		return errors.New("error storing CTFd access token in configmap: " + err.Error())
	}
	log.Println("CTFd access token stored in configmap")

	return nil
}

func getCTFdAccessToken() string {
	// Get the configmap
	configMap, err := getConfigMap(getNamespace(), CTFDACCESSCONFIGMAP)
	if err != nil {
		log.Println("Error getting CTFd access token configmap:", err)
		return ""
	}

	// Get the access token from the configmap
	accessToken := configMap.Data["access_token"]
	if accessToken == "" {
		log.Println("CTFd access token not found in configmap")
		return ""
	}

	return accessToken
}

func deleteCTFdPages(client *ctfd.Client) error {
	pages, err := client.GetPages(&ctfd.GetPagesParams{})
	if err != nil {
		return errors.New("error getting CTFd pages: " + err.Error())
	}
	for _, page := range pages {
		if page.ID == 0 {
			continue // Skip if ID is 0
		}
		IdStr := strconv.Itoa(page.ID)
		if err := client.DeletePage(IdStr); err != nil {
			return errors.New("error deleting CTFd page '" + IdStr + "': " + err.Error())
		}
		log.Printf("CTFd page '%s' deleted successfully", IdStr)
	}
	log.Println("CTFd pages deleted successfully")
	return nil
}

func getCTFdClient() (*ctfd.Client, error) {
	// Get the CTFd URL
	url := getCTFdURL()

	// Get the nonce and session
	nonce, session, err := ctfd.GetNonceAndSession(url)
	if err != nil {
		log.Println("Error getting nonce and session:", err)
		return nil, err
	}

	// Create a new CTFd client
	client := ctfd.NewClient(url, nonce, session, getCTFdAccessToken())

	return client, nil
}

func postSetupCTFd(w http.ResponseWriter, r *http.Request) {
	// Parse the request body
	var params CTFdSetupParams
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&params)
	if err != nil {
		errorResponse(w, r, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate the parameters
	if err := validateCTFdSetupParams(&params); err != nil {
		errorResponse(w, r, http.StatusBadRequest, err.Error())
		return
	}

	url := getCTFdURL()

	// Ensuring the CTFd is ready for setup
	// Call ctfd/setup
	res, err := (&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}).Get(url + "/setup")
	// Check for errors
	if err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to connect to CTFd: "+err.Error())
		return
	}
	defer res.Body.Close()
	// Check the response status code
	if res.StatusCode != http.StatusOK {
		errorResponse(w, r, http.StatusConflict, "CTFd is not ready for setup: "+res.Status)
		return
	}
	log.Println("CTFd is ready for setup - Got status code:", res.StatusCode)

	// Create a new CTFd client
	nonce, session, err := ctfd.GetNonceAndSession(url)
	if err != nil {
		errorResponse(w, r, http.StatusInternalServerError, "Failed to get nonce and session: "+err.Error())
		return
	}

	client := ctfd.NewClient(url, nonce, session, "")
	// Set up CTFd
	if err := setupCTFd(client, &params); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Set up CTFd brackets if provided
	if len(params.Brackets) > 0 {
		if err := setupCTFdBrackets(client, &params); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if params.MailServer != "" {
		// Set up CTFd mail settings
		if err := setupCTFdMailSettings(client, &params); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if params.RegistrationCode != "" {
		// Set up CTFd registration code
		if err := setupCTFdRegistrationCode(client, &params); err != nil {
			errorResponse(w, r, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := setupCTFdAccessToken(client, &params); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Delete existing CTFd pages
	if err := deleteCTFdPages(client); err != nil {
		errorResponse(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	// Respond with success
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{\"status\":\"success\"}"))
}
