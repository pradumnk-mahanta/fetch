package services

import (
	"context"
	"errors"
	"fetchtb/config"
	"fetchtb/databases"
	"os"
	"path/filepath"
	"sync"

	"github.com/forest6511/gdl"
)

type DownloadTask struct {
	ID         string
	URL        string
	OutputPath string

	Ctx        context.Context
	CancelFunc context.CancelFunc

	Stats  *gdl.DownloadStats
	Status string

	mu sync.Mutex
}

type GDLService struct {
	tasks map[string]*DownloadTask
	mu    sync.Mutex
}

func NewGDLService() *GDLService {
	return &GDLService{
		tasks: make(map[string]*DownloadTask),
	}
}

func (s *GDLService) Download(url string, category string, packageName string, dbID string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	dir, err := buildDownloadPath(category, packageName)
	if err != nil {
		return err
	}

	outputPath := filepath.Join(dir, packageName)

	task := &DownloadTask{
		ID:         dbID,
		URL:        url,
		OutputPath: outputPath,
		Ctx:        ctx,
		CancelFunc: cancel,
		Status:     config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING,
	}

	s.mu.Lock()
	s.tasks[dbID] = task
	s.mu.Unlock()

	databases.UpdateLocalDownloadStatus(dbID, config.DOWNLOAD_STATUS_DOWNLOADER_DOWNLOADING)

	go s.startDownload(task)

	return nil
}

func (s *GDLService) startDownload(task *DownloadTask) {
	options := &gdl.Options{
		EnableResume:      true,
		RetryAttempts:     1,
		MaxConcurrency:    2,
		CreateDirs:        true,
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			task.mu.Lock()
			defer task.mu.Unlock()
		},
	}

	stats, err := gdl.DownloadWithOptions(
		task.Ctx,
		task.URL,
		task.OutputPath,
		options,
	)

	task.mu.Lock()
	defer task.mu.Unlock()

	if err != nil {
		task.Status = config.DOWNLOAD_STATUS_CLIENT_FAILED
		databases.UpdateLocalDownloadStatus(task.ID, config.DOWNLOAD_STATUS_CLIENT_FAILED)
		return
	}

	task.Stats = stats
	task.Status = config.DOWNLOAD_STATUS_CLIENT_COMPLETED
	databases.UpdateLocalDownloadStatus(task.ID, config.DOWNLOAD_STATUS_CLIENT_COMPLETED)

	s.mu.Lock()
	delete(s.tasks, task.ID)
	s.mu.Unlock()
}

func (s *GDLService) Pause(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	s.mu.Unlock()

	if !exists {
		return errors.New("download not found")
	}

	task.CancelFunc()
	task.Status = config.DOWNLOAD_STATUS_DOWNLOADER_PAUSED
	databases.UpdateLocalDownloadStatus(id, config.DOWNLOAD_STATUS_DOWNLOADER_PAUSED)

	return nil
}

func (s *GDLService) Resume(id string) error {
	detail, err := databases.GetLocalDownloadDetails(id)
	if err != nil {
		return err
	}

	return s.Download(
		detail.OriginalDownloadURL,
		detail.Category,
		detail.DownloadName,
		id,
	)
}

func (s *GDLService) Delete(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if exists {
		task.CancelFunc()
		delete(s.tasks, id)
	}
	s.mu.Unlock()

	detail, err := databases.GetLocalDownloadDetails(id)
	if err == nil {
		root := os.Getenv("DOWNLOAD_ROOT")
		if root != "" {
			dir := filepath.Join(root, detail.Category, detail.DownloadName)
			_ = os.RemoveAll(dir)
		}
	}

	return databases.DeleteLocalDownload(id)
}

func buildDownloadPath(category string, packageName string) (string, error) {
	root := os.Getenv("DOWNLOAD_ROOT")
	if root == "" {
		return "", errors.New("DOWNLOAD_ROOT not set")
	}

	finalDir := filepath.Join(root, category, packageName)

	err := os.MkdirAll(finalDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return finalDir, nil
}
