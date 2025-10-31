package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/go-crypt/crypt"
	"github.com/go-crypt/crypt/algorithm"
	"github.com/go-crypt/crypt/algorithm/argon2"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

// --- Config Struct (Optional but recommended for type safety) ---
type Config struct {
	DBHost           string `mapstructure:"DB_HOST"`
	DBUser           string `mapstructure:"DB_USER"`
	DBPassword       string `mapstructure:"DB_PASSWORD"`
	DBName           string `mapstructure:"DB_NAME"`
	DBPort           string `mapstructure:"DB_PORT"`
	DBSSLMode        string `mapstructure:"DB_SSLMODE"`
	DBTimezone       string `mapstructure:"DB_TIMEZONE"`
	AppEnv           string `mapstructure:"APP_ENV"`
	Port             string `mapstructure:"PORT"`
	FrontendURL      string `mapstructure:"FRONTEND_URL"`
	UserVerification bool   `mapstructure:"FF_USER_VERIFICATION"`
}

var DB *gorm.DB
var AppConfig Config
var digest algorithm.Digest
var hasher *argon2.Hasher

// --- Entities ---

// Form represents the structure of a form
type Form struct {
	ID            uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` // Use DB generation for UUIDs
	Title         string         `json:"title" binding:"required"`
	Description   string         `json:"description"`
	CreatorUserID string         `json:"creator_user_id" binding:"required"` // Consider if this should be validated
	Questions     []Question     `json:"questions,omitempty" gorm:"foreignKey:FormID;constraint:OnDelete:CASCADE;"`
	CreatedAt     time.Time      `json:"created_at"` // Add explicitly
	UpdatedAt     time.Time      `json:"updated_at"` // Add explicitly
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

// Question represents a single question within a form
type Question struct {
	ID         uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` // Use DB generation for UUIDs
	FormID     uuid.UUID      `json:"-" gorm:"type:uuid"`                                       // Hide from JSON, ensure type match
	Text       string         `json:"text" binding:"required"`
	Type       string         `json:"type"` // Consider defining allowed types (e.g., text, rating, choice)
	IsRequired bool           `json:"is_required"`
	ExtraInfo  string         `json:"extra_info"`
	CreatedAt  time.Time      `json:"created_at"` // Add explicitly
	UpdatedAt  time.Time      `json:"updated_at"` // Add explicitly
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

type QuestionRequest struct {
	Text       string `json:"text" binding:"required"`
	Type       string `json:"type"`
	IsRequired bool   `json:"is_required"`
	ExtraInfo  string `json:"extra_info"`
}

// Answer represents a single answer to a question within a response
type Answer struct {
	ID         uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` // Use DB generation for UUIDs
	ResponseID uuid.UUID      `json:"-" gorm:"type:uuid"`                                       // Hide from JSON, ensure type match
	QuestionID uuid.UUID      `json:"question_id" binding:"required" gorm:"type:uuid"`          // Ensure type match
	Value      string         `json:"value"`                                                    // Consider max length or validation based on question type
	CreatedAt  time.Time      `json:"created_at"`                                               // Add explicitly
	UpdatedAt  time.Time      `json:"updated_at"`                                               // Add explicitly
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

// Response represents a submission for a specific form
type Response struct {
	ID               uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"` // Use DB generation for UUIDs
	FormID           uuid.UUID      `json:"form_id" gorm:"type:uuid"`                                 // Ensure type match
	RespondentUserID string         `json:"respondent_user_id" binding:"required"`                    // Consider if this should be validated
	Answers          []Answer       `json:"answers,omitempty" gorm:"foreignKey:ResponseID;constraint:OnDelete:CASCADE;"`
	CreatedAt        time.Time      `json:"created_at"` // Add explicitly
	UpdatedAt        time.Time      `json:"updated_at"` // Add explicitly
	DeletedAt        gorm.DeletedAt `json:"-" gorm:"index"`
}

type User struct {
	ID        uuid.UUID      `json:"id" gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Username  string         `json:"username" binding:"required"`
	Email     string         `json:"email" binding:"required"`
	Password  string         `json:"-" binding:"required"`
	Verified  bool           `json:"verified"`
	CreatedAt time.Time      `json:"created_at"` // Add explicitly
	UpdatedAt time.Time      `json:"updated_at"` // Add explicitly
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type Verification struct {
	gorm.Model
	UserID           uuid.UUID `json:"user_id" gorm:"type:uuid"`
	User             User      `json:"-" gorm:"foreignKey:UserID"`
	VerificationCode string    `json:"verification_code"`
	ExpiresAt        time.Time `json:"expires_at"`
}

type SignupRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// --- Load Configuration Function ---
func LoadConfig() {
	// 1. Load .env file (primarily for local development)
	// This attempts to load a .env file in the current directory.
	// It's okay if it fails (e.g., file not found in production),
	// as environment variables will take precedence later.
	err := godotenv.Load()
	if err != nil {
		// Log only if it's not a "file not found" error, or just log info level.
		if !os.IsNotExist(err) {
			log.Printf("Warning: Error loading .env file: %v", err)
		} else {
			log.Println("Info: .env file not found, relying on environment variables and defaults.")
		}
	}

	// 2. Configure Viper
	// Optional: Set config file name and paths if you want to use a config file (e.g., config.yaml)
	// viper.SetConfigName("config") // Name of config file (without extension)
	// viper.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	// viper.AddConfigPath(".")      // Look for config in the current directory
	// viper.AddConfigPath("./config") // Optionally look in a ./config folder

	// Optional: Attempt to read a configuration file if you decide to use one
	// if err := viper.ReadInConfig(); err != nil {
	// 	if _, ok := err.(viper.ConfigFileNotFoundError); ok {
	// 		// Config file not found; ignore error if desired
	// 		log.Println("Info: Configuration file not found.")
	// 	} else {
	// 		// Config file was found but another error was produced
	// 		log.Printf("Warning: Error reading config file: %v", err)
	// 	}
	// }

	// 3. Set Defaults (Crucial for fallback values)
	viper.SetDefault("DB_HOST", "localhost")
	viper.SetDefault("DB_USER", "postgres")
	viper.SetDefault("DB_PASSWORD", "") // Avoid defaulting sensitive data
	viper.SetDefault("DB_NAME", "gform_db")
	viper.SetDefault("DB_PORT", "5432")
	viper.SetDefault("DB_SSLMODE", "disable")
	viper.SetDefault("DB_TIMEZONE", "UTC") // Prefer UTC as a default
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("FRONTEND_URL", "frontend") // Default for dev
	viper.SetDefault("FF_USER_VERIFICATION", true)

	// 4. Enable reading from Environment Variables
	viper.AutomaticEnv() // Read in environment variables that match keys
	// Optional: If your env vars have a prefix like GFORM_DB_HOST
	// viper.SetEnvPrefix("GFORM")
	// Allows viper to read DB_HOST as DB.HOST if needed, useful for nested structs
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 5. Unmarshal the final configuration into the AppConfig struct
	// Viper applies precedence: Env Vars > Config File > Defaults
	err = viper.Unmarshal(&AppConfig)
	if err != nil {
		log.Fatalf("Unable to decode config into struct, %v", err)
	}

	// Optional: Add validation for critical missing configs like DB password in prod
	if AppConfig.AppEnv == "production" && AppConfig.DBPassword == "" {
		log.Fatalf("FATAL: DB_PASSWORD must be set in production environment!")
	}

	if hasher, err = argon2.New(argon2.WithProfileRFC9106LowMemory()); err != nil {
		panic(err)
	}

	log.Println("Configuration loaded successfully.")
	// For debugging purposes during development:
	// log.Printf("Debug: Loaded Config: %+v\n", AppConfig)
}

func NewDecoderArgon2idOnly() (decoder *crypt.Decoder, err error) {
	decoder = crypt.NewDecoder()

	if err = argon2.RegisterDecoderArgon2id(decoder); err != nil {
		return nil, err
	}

	return decoder, nil
}

// --- Database Functions ---

func ConnectDatabase() {
	connectionString := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		AppConfig.DBHost, AppConfig.DBUser, AppConfig.DBPassword, AppConfig.DBName,
		AppConfig.DBPort, AppConfig.DBSSLMode, AppConfig.DBTimezone)

	var err error
	DB, err = gorm.Open(postgres.Open(connectionString), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("Database connection successfully opened")

	sqlDB, err := DB.DB()
	if err != nil {
		log.Printf("Warning: Failed to get underlying database connection pool: %v", err)
		return
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
}

// AutoMigrateDatabase runs GORM's auto-migration feature.
func AutoMigrateDatabase() {
	log.Println("Starting database auto-migration...")
	// Ensure correct order if there are dependencies not handled by GORM automatically
	err := DB.AutoMigrate(
		&Form{},
		&Question{},
		&Response{},
		&Answer{},
		&User{},
		&Verification{},
	)

	if err != nil {
		log.Fatalf("Failed to auto-migrate database: %v", err)
	}
	log.Println("Database auto-migration completed")
}

// CreateForm saves a new form and its questions to the database.
func CreateForm(form *Form) error {
	// UUIDs for Form and Questions are now handled by the DB (default: gen_random_uuid())
	// GORM automatically handles associations if `form.Questions` is populated.
	result := DB.Create(form) // Create the form and its nested questions
	return result.Error
}

// GetFormByID retrieves a form and its questions by ID.
func GetFormByID(id string) (*Form, error) {
	var form Form
	// Preload fetches associated questions.
	// Use First to get a single record; returns ErrRecordNotFound if no match.
	result := DB.Preload("Questions").First(&form, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil // Standard way to indicate "not found"
		}
		return nil, result.Error // Other database error
	}
	return &form, nil
}

// GetAllForms retrieves all forms with their questions.
func GetAllFormsByUser(userId string) ([]Form, error) {
	var forms []Form
	// Use Find to get multiple records. Preload questions for each form.
	result := DB.Preload("Questions").
		Where("deleted_at IS NULL AND creator_user_id=?", userId).
		Order("created_at desc").Find(&forms)
	if result.Error != nil {
		return nil, result.Error
	}
	// Return empty slice if no forms found, not nil
	if forms == nil {
		forms = []Form{}
	}
	return forms, nil
}

func (form *Form) SetQuestions(questions []Question) error {
	form.Questions = questions
	return UpdateForm(form)
}

func (form *Form) AddQuestion(question *Question) error {
	if form.Questions == nil {
		form.Questions = []Question{}
	}

	form.Questions = append(form.Questions, *question)

	return UpdateForm(form)
}

// UpdateForm updates an existing form (and potentially its questions via associations).
func UpdateForm(form *Form) error {
	result := DB.Save(form)
	return result.Error
}

// DeleteForm deletes a form by ID. Associated questions/responses might be deleted by CASCADE constraint.
func DeleteForm(id string) error {
	// DB.Delete performs a soft delete if gorm.Model is used (sets deleted_at).
	// Use DB.Unscoped().Delete(...) for a hard delete.
	result := DB.Delete(&Form{}, "id = ?", id)
	// Check RowsAffected if you need to know if a record was actually deleted.
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound // Return not found if ID didn't exist
	}
	return result.Error
}

// CreateResponse saves a new response and its answers to the database.
func CreateResponse(response *Response) error {
	// UUIDs for Response and Answers are handled by the DB.
	// SubmittedAt is handled by gorm.Model's CreatedAt.
	// GORM automatically handles associations if `response.Answers` is populated.
	result := DB.Create(response)
	return result.Error
}

// GetResponseByID retrieves a response and its answers by ID.
func GetResponseByID(id string) (*Response, error) {
	var response Response
	result := DB.Preload("Answers").First(&response, "id = ?", id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, result.Error
	}
	return &response, nil
}

// GetResponsesByFormID retrieves all responses for a specific form ID.
func GetResponsesByFormID(formID string) ([]Response, error) {
	var responses []Response
	// Find responses where form_id matches, preload associated answers.
	result := DB.Preload("Answers").Where("form_id = ?", formID).Order("created_at desc").Find(&responses)
	if result.Error != nil {
		return nil, result.Error
	}
	// Return empty slice if no responses found, not nil
	if responses == nil {
		responses = []Response{}
	}
	return responses, nil
}

// GetAllResponses retrieves all responses (use with caution on large datasets).
func GetAllResponses() ([]Response, error) {
	var responses []Response
	result := DB.Preload("Answers").Order("created_at desc").Find(&responses)
	if result.Error != nil {
		return nil, result.Error
	}
	if responses == nil {
		responses = []Response{}
	}
	return responses, nil
}

// UpdateResponse updates an existing response.
func UpdateResponse(response *Response) error {
	result := DB.Save(response)
	return result.Error
}

// DeleteResponse deletes a response by ID. Associated answers might be deleted by CASCADE.
func DeleteResponse(id string) error {
	result := DB.Delete(&Response{}, "id = ?", id)
	if result.Error == nil && result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return result.Error
}

// Note: CRUD functions for individual Questions/Answers might be less common
// if they are always managed through their parent Form/Response,
// but can be added if needed (similar structure to Form/Response CRUD).

// --- Handlers ---

// createFormHandler handles POST /forms requests.
func createFormHandler(c *gin.Context) {
	var newForm Form

	session := sessions.Default(c)

	if session.Get("username") == nil {
		http.Redirect(c.Writer, c.Request, "/api/account/login", http.StatusSeeOther)
		return
	}

	// Bind JSON payload to the Form struct
	if err := c.ShouldBindJSON(&newForm); err != nil {
		log.Printf("Error binding JSON form: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or incomplete JSON request: " + err.Error()})
		return
	}

	// Optional: Set default question type if not provided in the request
	for i := range newForm.Questions {
		if newForm.Questions[i].Type == "" {
			newForm.Questions[i].Type = "text" // Set your desired default type
		}
		// DB will generate Question IDs
	}

	var userFound User
	DB.Find(&userFound, "username = ?", session.Get("username"))

	if userFound.ID == uuid.Nil {
		log.Printf("Error: User not found for session username %s", session.Get("username"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	newForm.CreatorUserID = userFound.ID.String()

	// Attempt to create the form in the database
	if err := CreateForm(&newForm); err != nil {
		log.Printf("Error creating form in DB: %v", err)
		// Provide a generic error message to the client
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save form"})
		return
	}

	// Log success and return the created form (now with DB-generated IDs and timestamps)
	log.Printf("Form created: ID=%s, Title=%s", newForm.ID, newForm.Title)
	c.JSON(http.StatusCreated, newForm)
}

func setQuestionsHandler(c *gin.Context) {
	formIDStr := c.Param("formId")
	var questionsRequest []QuestionRequest
	var questions []Question

	foundForm, err := GetFormByID(formIDStr)

	if err != nil {
		log.Printf("Error checking form %s existence for adding question: %v", formIDStr, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	if err := c.ShouldBindJSON(&questionsRequest); err != nil {
		log.Printf("Error binding JSON questions: %v", err)
		c.JSON(http.StatusBadRequest, "Invalid or incomplete JSON request: "+err.Error())
		return
	}

	if len(questionsRequest) > 50 {
		log.Printf("Error too many questions (max length is 50)")
		c.JSON(http.StatusBadRequest, "Too many questions (max length is 50)")
		return
	}

	tx := DB.Begin()
	if err := tx.Where("form_id = ?", formIDStr).Delete(&Question{}).Error; err != nil {
		tx.Rollback()
		return
	}

	for i := range questionsRequest {
		var question Question
		question.Text = questionsRequest[i].Text
		question.Type = questionsRequest[i].Type
		question.IsRequired = questionsRequest[i].IsRequired
		question.ExtraInfo = questionsRequest[i].ExtraInfo
		question.FormID = foundForm.ID
		question.CreatedAt = time.Now()
		question.UpdatedAt = time.Now()
		questions = append(questions, question)
	}

	if err := tx.Create(&questions).Error; err != nil {
		log.Printf("Error creating questions in DB: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save questions"})
		tx.Rollback()
		return
	}

	foundForm.Questions = questions // Update the form with the new questions

	tx.Commit()

	c.JSON(http.StatusOK, foundForm)

}

// listFormsHandler handles GET /forms requests.
func listFormsHandler(c *gin.Context) {

	session := sessions.Default(c)

	if session.Get("username") == nil {
		http.Redirect(c.Writer, c.Request, "/api/account/login", http.StatusSeeOther)
		return
	}

	var userFound User
	DB.Find(&userFound, "username = ?", session.Get("username"))

	if userFound.ID == uuid.Nil {
		log.Printf("Error: User not found for session username %s", session.Get("username"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "User not found"})
		return
	}

	allForms, err := GetAllFormsByUser(userFound.ID.String())
	if err != nil {
		log.Printf("Error retrieving forms: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving forms"})
		return
	}

	// Return the list of forms (will be an empty array [] if none found)
	c.JSON(http.StatusOK, allForms)
}

// getFormHandler handles GET /forms/:formId requests.
func getFormHandler(c *gin.Context) {
	formID := c.Param("formId")

	// Validate if formID looks like a UUID (optional but good practice)
	// if _, err := uuid.Parse(formID); err != nil {
	// 	c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID format"})
	//  return
	// }

	form, err := GetFormByID(formID)
	if err != nil {
		// Log the actual database error
		log.Printf("Error retrieving form %s: %v", formID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving form"})
		return
	}
	// GetFormByID returns nil, nil if not found
	if form == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	c.JSON(http.StatusOK, form)
}

// submitResponseHandler handles POST /forms/:formId/responses requests.
func submitResponseHandler(c *gin.Context) {

	formIDStr := c.Param("formId")

	// Attempt to parse the string into a UUID
	formID, err := uuid.Parse(formIDStr)
	if err != nil {
		// The string was not a valid UUID format
		// Return a Bad Request error to the client
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid form ID format"})
		return // Stop further execution in the handler
	}

	// 1. Check if the target form exists
	targetForm, err := GetFormByID(formIDStr)
	if err != nil {
		log.Printf("Error checking form %s existence for submission: %v", formIDStr, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving form"})
		return
	}
	if targetForm == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	// 2. Bind the incoming response data
	var newResponse Response
	if err := c.ShouldBindJSON(&newResponse); err != nil {
		log.Printf("Error binding JSON response: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or incomplete JSON request: " + err.Error()})
		return
	}

	// 3. Validate the response against the form's questions
	questionMap := make(map[uuid.UUID]Question) // Map question ID to Question struct for easy lookup
	for _, q := range targetForm.Questions {
		questionMap[q.ID] = q
	}

	answeredQuestions := make(map[uuid.UUID]bool) // Track answered question IDs
	for i := range newResponse.Answers {
		ans := &newResponse.Answers[i] // Get pointer to modify if needed (e.g., setting ResponseID)

		// Check if the QuestionID submitted actually exists in the target form
		q, exists := questionMap[ans.QuestionID]
		if !exists {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid question ID in response: " + ans.QuestionID.String()})
			return
		}

		// Check if a required question was left empty
		// Note: Allows empty string for non-required questions
		if ans.Value == "" && q.IsRequired {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing answer for required question: " + q.Text})
			return
		}
		answeredQuestions[ans.QuestionID] = true
		// DB will generate Answer IDs
		// The ResponseID will be set automatically by GORM when creating the Response with nested Answers
	}

	// Check if all required questions from the form were answered
	for _, q := range targetForm.Questions {
		if q.IsRequired {
			if _, answered := answeredQuestions[q.ID]; !answered {
				c.JSON(http.StatusBadRequest, gin.H{"error": "Missing answer for required question: " + q.Text})
				return
			}
		}
	}

	// 4. Prepare and save the response
	newResponse.FormID = formID // Associate response with the form
	// ID and SubmittedAt (CreatedAt) will be handled by DB/GORM

	if err := CreateResponse(&newResponse); err != nil {
		log.Printf("Error creating response in DB for form %s: %v", formID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not save response"})
		return
	}

	log.Printf("Response submitted for Form ID=%s by UserID=%s, ResponseID=%s", formID, newResponse.RespondentUserID, newResponse.ID)
	// Return the created response (with DB-generated IDs/timestamps)
	c.JSON(http.StatusCreated, newResponse)
}

// getFormResponsesHandler handles GET /forms/:formId/responses requests.
func getFormResponsesHandler(c *gin.Context) {
	formID := c.Param("formId")

	// Optional: Check if form exists first to give a specific 404 for the form
	// form, err := GetFormByID(formID) // Re-uses the function
	// if err != nil {
	// 	log.Printf("Error checking form %s existence for getting responses: %v", formID, err)
	// 	c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving form"})
	// 	return
	// }
	// if form == nil {
	// 	c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
	// 	return
	// }
	// If you skip the check above, GetResponsesByFormID will just return an empty list if the formID doesn't exist.

	// Retrieve responses associated with the form ID
	responses, err := GetResponsesByFormID(formID)
	if err != nil {
		log.Printf("Error retrieving responses for form %s: %v", formID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error retrieving responses"})
		return
	}

	// Return the list of responses (will be an empty array [] if none found)
	c.JSON(http.StatusOK, responses)
}

// --- Authentication Handlers --- //
func signupHandler(c *gin.Context) {
	var signupRequest SignupRequest
	if err := c.ShouldBindJSON(&signupRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var count int64 = 0
	DB.Find(&User{}, "email = ?", signupRequest.Email).Count(&count)

	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email already exists"})
		return
	}

	DB.Find(&User{}, "username = ?", signupRequest.Username).Count(&count)
	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username already exists"})
		return
	}

	var err error
	if digest, err = hasher.Hash(signupRequest.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var newUser User
	newUser.Username = signupRequest.Username
	newUser.Email = signupRequest.Email
	newUser.Password = digest.Encode()

	if ret := DB.Create(&newUser); ret.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": ret.Error})
		return
	}

	var newUserVerification Verification
	newUserVerification.UserID = newUser.ID
	newUserVerification.VerificationCode = uuid.New().String()
	newUserVerification.ExpiresAt = time.Now().Add(150 * time.Hour) // 24 hours

	if ret := DB.Create(&newUserVerification); ret.Error != nil {
		log.Println("Error creating verification record:", ret.Error)
	}

	log.Printf("Account %s created and can be verified at http://%s/api/account/verify?verificationCode=%s",
		newUser.Username,
		c.Request.Host,
		newUserVerification.VerificationCode)

	c.JSON(http.StatusCreated, newUser)
}

func signinHandler(c *gin.Context) {
	var loginRequest SignupRequest

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user User
	DB.Find(&user, "email = ?", loginRequest.Email)

	if user.ID == uuid.Nil {
		c.JSON(http.StatusForbidden, gin.H{"error": "Wrong credentials"})
		return
	}

	if AppConfig.UserVerification {
		if user.Verified == false {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account not verified"})
			return
		}
	}

	var valid bool
	var err error
	if valid, err = crypt.CheckPassword(loginRequest.Password, user.Password); err != nil || !valid {
		c.JSON(http.StatusForbidden, gin.H{"error": "Wrong credentials"})
		return
	}

	session := sessions.Default(c)
	session.Set("username", user.Username)
	session.Save()

	c.JSON(http.StatusOK, gin.H{"username": user.Username, "email": user.Email, "message": "Signed in successfully"})
}

func logoutHandler(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
}

func verifyHandler(c *gin.Context) {
	verificationParam := c.Query("verificationCode")

	log.Printf("Received verification code: %s", verificationParam)

	var verification Verification
	DB.Find(&verification, "verification_code = ?", verificationParam)

	log.Printf("Verification record found: %+v", verification)

	if verification.ID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Invalid verification code"})
		return
	}

	if time.Now().After(verification.ExpiresAt) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification code has expired"})
		return
	}

	// Fetch the user from the database to ensure we're working with the latest data
	var user User
	if err := DB.First(&user, verification.UserID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch user data"})
		return
	}

	user.Verified = true
	DB.Save(&user)
	DB.Delete(&verification) // Delete the verification record after successful verification
}

func whoamiHandler(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Not signed in"})
		return
	}

	var user User
	DB.Find(&user, "username = ?", username)

	if user.ID == uuid.Nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"username": user.Username, "email": user.Email, "message": "User details retrieved successfully"})
}

// --- Main Application Setup ---

func main() {
	// Load configuration FIRST
	LoadConfig()

	// Initialize database connection first
	ConnectDatabase()

	// Run migrations after connection is established
	AutoMigrateDatabase()

	// Set Gin mode based on environment (production/debug)
	appEnv := os.Getenv("APP_ENV")
	if appEnv == "production" || appEnv == "prod" {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode) // Default is debug
	}

	// Initialize Gin router
	router := gin.Default() // Includes logger and recovery middleware

	store := cookie.NewStore([]byte("secret"))
	router.Use(sessions.Sessions("gform_session", store))

	// --- CORS Configuration ---
	config := cors.DefaultConfig()
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}          // Include OPTIONS for preflight requests
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"} // Add any custom headers your frontend sends
	config.AllowCredentials = true                                                              // If your frontend needs to send cookies or auth headers

	// Dynamically set allowed origins based on environment
	if gin.Mode() == gin.ReleaseMode {
		frontendURL := os.Getenv("FRONTEND_URL") // e.g., https://your-app.com
		if frontendURL == "" {
			log.Println("WARNING: FRONTEND_URL environment variable not set for production mode! Allowing all origins (unsafe).")
			config.AllowAllOrigins = true // Use with caution! Prefer specific origin.
		} else {
			log.Printf("Production Mode: CORS enabled for %s", frontendURL)
			config.AllowOrigins = []string{frontendURL}
		}
	} else {
		// Development mode: Allow common local development origins
		log.Println("Development Mode: CORS enabled for localhost and 127.0.0.1 (ports 4200, 5173)")
		config.AllowOrigins = []string{"http://localhost", "http://localhost:4200", "http://127.0.0.1:4200", "http://localhost:5173", "http://127.0.0.1:5173"} // Add others if needed
	}

	router.Use(cors.New(config)) // Apply CORS middleware

	// --- API Routes ---
	// Group routes under /forms
	formRoutes := router.Group("/forms")
	{
		formRoutes.POST("", createFormHandler)                    // POST /forms
		formRoutes.PUT("/:formId/questions", setQuestionsHandler) // PUT /forms/{formId}/questions")
		formRoutes.GET("", listFormsHandler)                      // GET /forms
		formRoutes.GET("/:formId", getFormHandler)                // GET /forms/{formId}
		// Add PUT /forms/{formId} and DELETE /forms/{formId} handlers if needed

		// Group response routes under /forms/{formId}/responses
		responseRoutes := formRoutes.Group("/:formId/responses")
		{
			responseRoutes.POST("", submitResponseHandler)  // POST /forms/{formId}/responses
			responseRoutes.GET("", getFormResponsesHandler) // GET /forms/{formId}/responses
			// Add GET /forms/{formId}/responses/{responseId}, PUT, DELETE handlers if needed
		}

		authnRoutes := router.Group("/api/account")
		{
			authnRoutes.POST("/signup", signupHandler)
			authnRoutes.POST("/signin", signinHandler)
			authnRoutes.POST("/signout", logoutHandler)
			authnRoutes.POST("/verify", verifyHandler)
			authnRoutes.GET("/whoami", whoamiHandler)
		}

	}

	// --- Start Server ---
	port := os.Getenv("PORT") // Allow port configuration via environment variable
	if port == "" {
		port = "8080" // Default port
	}

	log.Printf("Gin server starting in '%s' mode on port :%s", gin.Mode(), port)
	// Use ":" prefix for binding to all network interfaces on that port
	err := router.Run(":" + port)
	if err != nil {
		log.Fatalf("Error starting Gin server: %v", err)
	}
}
