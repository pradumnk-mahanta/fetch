package services

import (
	"context"
	"errors"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/forest6511/gdl"
)

var (
	serviceInstance *GDLService
	once            sync.Once
)

func InitGDLService() {
	once.Do(func() {
		logger.Log.Infow("Initializing GDLService...")
		serviceInstance = &GDLService{
			tasks: make(map[string]*DownloadTask),
		}
	})
}

type DownloadTask struct {
	ID         string
	DownloadID string //Parent
	URL        string
	OutputPath string

	Ctx        context.Context
	CancelFunc context.CancelFunc

	Stats  *gdl.DownloadStats
	Status string

	mu sync.Mutex
}

func GetGDLService() *GDLService {
	if serviceInstance == nil {
		logger.Log.Errorw("GDLService not initialized. Call InitGDLService() first.")
		panic("GDLService not initialized")
	}
	return serviceInstance
}

type GDLService struct {
	tasks map[string]*DownloadTask
	mu    sync.Mutex
}

func (s *GDLService) Download(parentCtx context.Context, localDownloadItem models.LocalDownloadInstanceItem) error {
	ctx, cancel := context.WithCancel(parentCtx)

	localDownload, err := databases.GetLocalDownloadDetails(localDownloadItem.DownloadID)
	if err != nil {
		cancel()
		logger.Log.Errorw("Failed to get local download item", "error", err, "filename", localDownloadItem.FileName)
		return err
	}

	databases.UpdateLocalDownloadItemStatus(localDownloadItem.ID, localDownloadItem.DownloadID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING)

	dir, err := GetDownloadRootPath(localDownload.Category)
	if err != nil {
		cancel()
		logger.Log.Errorw("Failed to build download path", "error", err, "category", localDownload.Category, "packageName", localDownload.DownloadName)
		return err
	}

	outputPath := filepath.Join(dir, localDownloadItem.FilePath)
	task := &DownloadTask{
		ID:         localDownloadItem.ID,
		DownloadID: localDownloadItem.DownloadID,
		URL:        localDownloadItem.ExternalProviderDownloadURL,
		OutputPath: outputPath,
		Ctx:        ctx,
		CancelFunc: cancel,
		Status:     config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING,
		Stats: &gdl.DownloadStats{
			TotalSize: localDownloadItem.FileSize,
		},
	}

	s.mu.Lock()
	if _, exists := s.tasks[task.ID]; exists {
		s.mu.Unlock()
		cancel()
		return errors.New("Download already exists")
	}
	s.tasks[task.ID] = task
	s.mu.Unlock()

	go s.startDownload(task)

	return nil
}

func (s *GDLService) startDownload(task *DownloadTask) {
	options := &gdl.Options{
		EnableResume:      true,
		RetryAttempts:     1,
		MaxConcurrency:    4,
		CreateDirs:        true,
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			task.mu.Lock()
			task.Stats = &gdl.DownloadStats{
				BytesDownloaded: p.BytesDownloaded,
				TotalSize:       p.TotalSize,
				AverageSpeed:    p.Speed,
			}
			task.mu.Unlock()
		},
	}

	stats, err := gdl.DownloadWithOptions(
		task.Ctx,
		task.URL,
		task.OutputPath,
		options,
	)

	if err != nil {
		task.mu.Lock()

		if errors.Is(err, context.Canceled) {
			task.Status = config.DOWNLOAD_STATUS_DOWNLOADER_PAUSED
			task.mu.Unlock()

			databases.UpdateLocalDownloadItemStatus(task.ID, task.DownloadID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PAUSED)
			logger.Log.Infow("Download paused", "id", task.ID)

			s.mu.Lock()
			delete(s.tasks, task.ID)
			s.mu.Unlock()

			return
		}

		task.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED
		task.mu.Unlock()

		databases.UpdateLocalDownloadItemStatus(task.ID, task.DownloadID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED)
		logger.Log.Errorw("Failed to download", "error", err, "url", task.URL, "outputPath", task.OutputPath)

		s.mu.Lock()
		delete(s.tasks, task.ID)
		s.mu.Unlock()

		return
	}

	task.mu.Lock()
	task.Stats = stats
	task.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	task.mu.Unlock()

	databases.UpdateLocalDownloadItemStatus(task.ID, task.DownloadID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED)
	logger.Log.Infow("Download completed", "id", task.ID, "url", task.URL, "outputPath", task.OutputPath)

	s.mu.Lock()
	delete(s.tasks, task.ID)
	s.mu.Unlock()
}

func (s *GDLService) Pause(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if exists {
		task.CancelFunc()
	}
	s.mu.Unlock()

	if !exists {
		return errors.New("download not found")
	}

	task.CancelFunc()
	return nil
}

func (s *GDLService) Resume(prntCtx context.Context, id string) error {
	s.mu.Lock()
	_, running := s.tasks[id]
	s.mu.Unlock()

	if running {
		return errors.New("Download already running")
	}

	detail, err := databases.GetLocalDownloadItemDetails(id)
	if err != nil {
		return err
	}

	return s.Download(prntCtx, detail)
}

func (s *GDLService) Delete(id string) error {
	s.mu.Lock()
	task, exists := s.tasks[id]
	if exists {
		task.CancelFunc()
		delete(s.tasks, id)
	}
	s.mu.Unlock()

	downloadItem, err := databases.GetLocalDownloadItemDetails(id)
	if err == nil {
		download, err2 := databases.GetLocalDownloadDetails(downloadItem.DownloadID)
		if err2 == nil {
			categoryDir, _ := GetDownloadRootPath(download.Category)
			filePath := filepath.Join(categoryDir, downloadItem.FilePath)
			cleanPath := filepath.Clean(filePath)

			if strings.HasPrefix(cleanPath, categoryDir+string(os.PathSeparator)) {
				_ = os.Remove(cleanPath)
			}
		}
	}

	return nil
}

func (s *GDLService) Status() []models.GDLDownload {
	var statuses []models.GDLDownload

	s.mu.Lock()
	tasks := make([]*DownloadTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		tasks = append(tasks, t)
	}
	s.mu.Unlock()

	for _, task := range tasks {
		task.mu.Lock()

		status := models.GDLDownload{
			ID:              task.ID,
			URL:             task.URL,
			OutputPath:      task.OutputPath,
			Status:          task.Status,
			BytesDownloaded: 0,
			Percentage:      0,
			AverageSpeed:    "0 MB/s",
			TotalSize:       0,
		}

		if task.Stats != nil {
			status.BytesDownloaded = task.Stats.BytesDownloaded
			status.TotalSize = task.Stats.TotalSize
			if task.Stats.TotalSize > 0 {
				status.Percentage = (float64(task.Stats.BytesDownloaded) / float64(task.Stats.TotalSize)) * 100
			}
			status.AverageSpeed = fmt.Sprintf("%.2f MB/s", float64(task.Stats.AverageSpeed)/1024/1024)
		}

		statuses = append(statuses, status)

		task.mu.Unlock()
	}

	return statuses
}

func GetDownloadRootPath(category string) (string, error) {
	root := config.APPLICATION_DOWNLOAD_ROOT.GetValue()
	if root == "" {
		return "", errors.New("DOWNLOAD_ROOT not set")
	}

	finalDir := filepath.Join(root, category)

	err := os.MkdirAll(finalDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return finalDir, nil
}
