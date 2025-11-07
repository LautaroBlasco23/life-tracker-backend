package imagestore

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	pb "github.com/lautaroblasco23/imagestore/proto/imagestore/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	conn   *grpc.ClientConn
	client pb.ImageServiceClient
}

func NewClient(address string) (*Client, error) {
	conn, err := grpc.NewClient(
		address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create imagestore client: %w", err)
	}

	return &Client{
		conn:   conn,
		client: pb.NewImageServiceClient(conn),
	}, nil
}

func (c *Client) UploadProfileImage(ctx context.Context, userID uint, imageData []byte, filename string) (string, string, error) {
	stream, err := c.client.UploadImage(ctx)
	if err != nil {
		return "", "", fmt.Errorf("failed to create upload stream: %w", err)
	}

	contentType := getContentType(filename)
	metadata := &pb.ImageMetadataInput{
		UserId:      fmt.Sprintf("%d", userID),
		Filename:    filename,
		ContentType: contentType,
	}

	if err = stream.Send(&pb.UploadImageRequest{
		Data: &pb.UploadImageRequest_Metadata{
			Metadata: metadata,
		},
	}); err != nil {
		return "", "", fmt.Errorf("failed to send metadata: %w", err)
	}

	const chunkSize = 64 * 1024
	for i := 0; i < len(imageData); i += chunkSize {
		end := i + chunkSize
		if end > len(imageData) {
			end = len(imageData)
		}

		if err = stream.Send(&pb.UploadImageRequest{
			Data: &pb.UploadImageRequest_Chunk{
				Chunk: imageData[i:end],
			},
		}); err != nil {
			return "", "", fmt.Errorf("failed to send chunk: %w", err)
		}
	}

	resp, err := stream.CloseAndRecv()
	if err != nil {
		return "", "", fmt.Errorf("failed to receive response: %w", err)
	}

	return resp.Url, resp.ThumbnailUrl, nil
}

func (c *Client) DeleteImage(ctx context.Context, imageID string) error {
	_, err := c.client.DeleteImage(ctx, &pb.DeleteImageRequest{
		ImageId: imageID,
	})
	return err
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func getContentType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	contentTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".webp": "image/webp",
	}

	if ct, ok := contentTypes[ext]; ok {
		return ct
	}
	return "application/octet-stream"
}
