package infrastructure_test

import (
	"A2SV_Starter_Project_Blog/Domain"
	"A2SV_Starter_Project_Blog/Infrastructure"
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/suite"
)

// CloudinaryIntegrationSuite defines the test suite for the real Cloudinary service.
type CloudinaryIntegrationSuite struct {
	suite.Suite
	service   domain.ImageUploaderService
	cloudName string
}

// SetupSuite runs ONCE before any tests in the suite.
// It's the perfect place to check for credentials and skip if they're not available.
func (s *CloudinaryIntegrationSuite) SetupSuite() {
	godotenv.Load()
	godotenv.Load("../.env")
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	// This is a crucial step for CI/CD environments.
	// If credentials are not set, we skip the entire test suite.
	if cloudName == "" || apiKey == "" || apiSecret == "" {
		s.T().Skip("Skipping Cloudinary integration tests: environment variables not set")
	}

	s.cloudName = cloudName // Store for later assertions

	// Initialize the REAL service with the REAL credentials.
	var err error
	s.service, err = infrastructure.NewCloudinaryService(cloudName, apiKey, apiSecret)
	s.Require().NoError(err, "Failed to initialize real Cloudinary service")
}

// TestCloudinaryIntegrationSuite is the entry point for running the test suite.
func TestCloudinaryIntegrationSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode.")
	}
	suite.Run(t, new(CloudinaryIntegrationSuite))
}

// TestUploadProfilePicture_Integration is our actual integration test.
func (s *CloudinaryIntegrationSuite) TestUploadProfilePicture_Integration() {
	// Arrange: Create a realistic multipart form in memory to get a proper file handle.

	// 1. Create a buffer to hold the multipart data.
	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	// 2. Create a "form file" part. This returns an io.Writer.
	part, err := writer.CreateFormFile("file", "integration_test.jpg")
	s.Require().NoError(err, "Should be able to create form file")

	// 3. Write the actual content of our "file" into the part.
	fileContent := "this is the real content of the test file"
	_, err = io.WriteString(part, fileContent)
	s.Require().NoError(err, "Should be able to write to form file part")

	// 4. IMPORTANT: Close the writer to finalize the multipart message (adds boundaries).
	err = writer.Close()
	s.Require().NoError(err, "Should be able to close multipart writer")

	// 5. Now, parse this in-memory form to get a "real" multipart.File and *multipart.FileHeader
	// that the Cloudinary SDK will understand.
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	err = req.ParseMultipartForm(10 << 20) // 10 MB max memory
	s.Require().NoError(err, "Should be able to parse the created multipart form")

	file, header, err := req.FormFile("file")
	s.Require().NoError(err, "Should be able to extract the file from the parsed form")
	defer file.Close()

	// Act: Call the method with the properly constructed file and header.
	// This will now make a REAL network call with a REAL file.
	imageURL, err := s.service.UploadProfilePicture(file, header)

	// Assert: Check the results from the real API.
	s.NoError(err, "UploadProfilePicture should not return an error with valid credentials")
	s.NotEmpty(imageURL, "Returned image URL should not be empty")

	// Verify the URL structure is correct.
	s.Contains(imageURL, "https://res.cloudinary.com/", "URL should be a secure Cloudinary URL")
	s.Contains(imageURL, s.cloudName, "URL should contain the correct cloud name")
	s.Contains(imageURL, "/profile_pictures/", "URL should contain the correct folder")
}
