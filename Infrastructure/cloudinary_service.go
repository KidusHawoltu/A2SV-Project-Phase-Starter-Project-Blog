package infrastructure

import (
	domain "A2SV_Starter_Project_Blog/Domain"
	"context"
	"log"
	"mime/multipart"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type ClodinaryService struct {
	cld *cloudinary.Cloudinary
}

func NewCloudinaryService(cloudinaryCloudName, cloudinaryApiKey, cloudinaryApiSecret string) (domain.ImageUploaderService, error) {
	cld, err := cloudinary.NewFromParams(
		cloudinaryCloudName,
		cloudinaryApiKey,
		cloudinaryApiSecret,
	)
	if err != nil {
		log.Fatalf("Failed to initialize Cloudinary: %v", err)
	}
	return &ClodinaryService{cld: cld}, err
}

func NewCloudinaryServiceFromURL(url string) (domain.ImageUploaderService, error) {
	cld, err := cloudinary.NewFromURL(url)
	if err != nil {
		return nil, err
	}
	return &ClodinaryService{cld: cld}, nil
}

func (cs *ClodinaryService) UploadProfilePicture(file multipart.File, fileHeader *multipart.FileHeader) (string, error) {
	ctx := context.Background()

	uploadResult, err := cs.cld.Upload.Upload(ctx, file, uploader.UploadParams{
		Folder: "profile_pictures",
	})

	if err != nil {
		return "", err
	}

	return uploadResult.SecureURL, nil
}
