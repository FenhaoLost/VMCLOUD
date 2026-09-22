package config

import (
	"errors"
)

// MetricSample is a persisted per-container metric sample.
type MetricSample struct {
	TS        int64   `json:"ts"`
	CPU       float64 `json:"cpu"`
	Memory    float64 `json:"memory"`
	NetworkRx float64 `json:"network_rx"`
	NetworkTx float64 `json:"network_tx"`
	DiskRead  float64 `json:"disk_read"`
	DiskWrite float64 `json:"disk_write"`
}

// SaveMetricSamples persists metric samples for a container key.
func SaveMetricSamples(containerKey string, samples []MetricSample) error {
	if len(samples) == 0 {
		return nil
	}
	dbMu.Lock()
	defer dbMu.Unlock()
	if db == nil {
		return errors.New("database not initialized")
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO container_metrics
		(container_key, ts, cpu, memory, network_rx, network_tx, disk_read, disk_write)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer stmt.Close()
	for _, s := range samples {
		if _, err := stmt.Exec(containerKey, s.TS, s.CPU, s.Memory, s.NetworkRx, s.NetworkTx, s.DiskRead, s.DiskWrite); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// LoadMetricSamples returns metric samples for a container key at or after sinceMs.
func LoadMetricSamples(containerKey string, sinceMs int64) ([]MetricSample, error) {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db == nil {
		return nil, nil
	}
	rows, err := db.Query(`SELECT ts, cpu, memory, network_rx, network_tx, disk_read, disk_write
		FROM container_metrics WHERE container_key = ? AND ts >= ? ORDER BY ts ASC`, containerKey, sinceMs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	samples := make([]MetricSample, 0, 256)
	for rows.Next() {
		var s MetricSample
		if err := rows.Scan(&s.TS, &s.CPU, &s.Memory, &s.NetworkRx, &s.NetworkTx, &s.DiskRead, &s.DiskWrite); err != nil {
			return nil, err
		}
		samples = append(samples, s)
	}
	return samples, rows.Err()
}

// PruneMetricSamples deletes samples older than olderThanMs.
func PruneMetricSamples(olderThanMs int64) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM container_metrics WHERE ts < ?`, olderThanMs)
	return err
}

// PruneMetricSamplesForContainers deletes samples whose container no longer exists.
func PruneMetricSamplesForContainers(validKeys map[string]bool) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if db == nil {
		return nil
	}
	if len(validKeys) == 0 {
		return nil
	}
	args := make([]interface{}, 0, len(validKeys))
	placeholders := ""
	for key := range validKeys {
		if placeholders != "" {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, key)
	}
	query := `DELETE FROM container_metrics WHERE container_key NOT IN (` + placeholders + `)`
	_, err := db.Exec(query, args...)
	return err
}
