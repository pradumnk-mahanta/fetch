package services

import (
	"archive/zip"
	"context"
	"errors"
	"fetch/config"
	"fetch/databases"
	"fetch/logger"
	"fetch/models"
	"fmt"
	"io"
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

func (s *GDLService) Download(parentCtx context.Context, localDownloadItem databases.LocalDownloadsInstanceItem) error {
	ctx, cancel := context.WithCancel(parentCtx)

	databases.UpdateLocalDownloadItemStatus(localDownloadItem.IDString(), config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING)

	dir, err := GetDownloadRootPath(localDownloadItem.Category)
	if err != nil {
		cancel()
		logger.Log.Errorw("Failed to build download path", "error", err, "category", localDownloadItem.Category, "packageName", localDownloadItem.FileName)
		return err
	}

	outputPath := filepath.Join(dir, localDownloadItem.FilePath)
	task := &DownloadTask{
		ID:         localDownloadItem.IDString(),
		DownloadID: localDownloadItem.DownloadIDString(),
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
			logger.Log.Debugw("Download Progress", "Id", task.ID, "File", task.OutputPath, "Percentage", p.Percentage)
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

			databases.UpdateLocalDownloadItemStatus(task.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PAUSED)
			logger.Log.Infow("Download paused", "id", task.ID)

			s.mu.Lock()
			delete(s.tasks, task.ID)
			s.mu.Unlock()

			return
		}

		task.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED
		task.mu.Unlock()

		databases.UpdateLocalDownloadItemStatus(task.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED)
		logger.Log.Errorw("Failed to download", "error", err, "url", task.URL, "outputPath", task.OutputPath)

		s.mu.Lock()
		delete(s.tasks, task.ID)
		s.mu.Unlock()

		return
	}

	task.mu.Lock()
	task.Stats = stats
	task.mu.Unlock()

	downloadItem, err := databases.GetLocalDownloadItemDetails(task.ID)
	if err == nil && downloadItem.DownloadType == config.DOWNLOAD_ITEM_TYPE_FULL_ARCHIVE {
		databases.UpdateLocalDownloadItemStatus(task.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_PROCESSING)
		logger.Log.Infow("Extracting archive", "id", task.ID, "path", task.OutputPath)
		extractDir := filepath.Dir(task.OutputPath)
		if err := extractZip(task.OutputPath, extractDir); err != nil {
			logger.Log.Errorw("Failed to extract archive", "error", err, "id", task.ID)

			task.mu.Lock()
			task.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED
			task.mu.Unlock()

			databases.UpdateLocalDownloadItemStatus(task.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_FAILED)

			s.mu.Lock()
			delete(s.tasks, task.ID)
			s.mu.Unlock()

			return
		}
		_ = os.Remove(task.OutputPath)
	}

	task.mu.Lock()
	task.Status = config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED
	task.mu.Unlock()

	databases.UpdateLocalDownloadItemStatus(task.ID, config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_COMPLETED)

	logger.Log.Infow("Download completed", "id", task.ID)
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
		return errors.New("Download already running!")
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
			AverageSpeed:    0,
			TotalSize:       0,
		}

		if task.Stats != nil {
			status.BytesDownloaded = task.Stats.BytesDownloaded
			status.TotalSize = task.Stats.TotalSize
			if task.Stats.TotalSize > 0 {
				status.Percentage = (float64(task.Stats.BytesDownloaded) / float64(task.Stats.TotalSize)) * 100
			}
			status.AverageSpeed = float64(task.Stats.AverageSpeed) / 1024
		}

		statuses = append(statuses, status)

		task.mu.Unlock()
	}

	return statuses
}

func GetDownloadRootPath(category string) (string, error) {
	root := config.ApplicationDownloadRoot
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

func (s *GDLService) IsDownloading(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists || task.Status != config.DOWNLOAD_ITEM_STATUS_DOWNLOADER_DOWNLOADING {
		return false
	}

	return true
}

func extractZip(source, destination string) error {
	r, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(destination, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destination)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in zip: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, os.ModePerm); err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)

		outFile.Close()
		rc.Close()

		if err != nil {
			return err
		}
	}
	return nil
}
