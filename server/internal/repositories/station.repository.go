package repositories

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/tylerjvollick/nori/internal/models"
	"gorm.io/gorm"
)

type StationRepository struct {
	db *gorm.DB
}

func NewStationRepository(db *gorm.DB) *StationRepository {
	return &StationRepository{db: db}
}

func (r *StationRepository) Create(station *models.Station) error {
	return r.db.Create(station).Error
}

func (r *StationRepository) GetByID(id uuid.UUID) (*models.Station, error) {
	var station models.Station
	err := r.db.First(&station, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("station not found")
		}
		return nil, err
	}
	return &station, nil
}

func (r *StationRepository) GetBySpaceID(spaceID uuid.UUID) ([]models.Station, error) {
	var stations []models.Station
	err := r.db.Where("space_id = ?", spaceID).
		Order("display_order ASC").
		Find(&stations).Error
	return stations, err
}

func (r *StationRepository) Update(station *models.Station) error {
	return r.db.Save(station).Error
}

func (r *StationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&models.Station{}, "id = ?", id).Error
}

// Reorder updates the display_order of all stations for a given space.
// The ids slice defines the new order — ids[0] gets display_order 0, etc.
func (r *StationRepository) Reorder(spaceID uuid.UUID, ids []uuid.UUID) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for i, id := range ids {
			err := tx.Model(&models.Station{}).
				Where("id = ? AND space_id = ?", id, spaceID).
				Update("display_order", i).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// WIPCount holds the WIP count for a single station.
type WIPCount struct {
	StationID uuid.UUID
	Count     int
}

// GetWIPCounts returns the number of active or paused tasks at each station in a space.
func (r *StationRepository) GetWIPCounts(spaceID uuid.UUID) (map[uuid.UUID]int, error) {
	var rows []WIPCount
	err := r.db.Model(&models.Task{}).
		Select("station_id, COUNT(*) as count").
		Where("space_id = ? AND station_id IS NOT NULL AND status IN ?", spaceID, []string{"active", "paused"}).
		Group("station_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]int, len(rows))
	for _, row := range rows {
		result[row.StationID] = row.Count
	}
	return result, nil
}

// GetMaxDisplayOrder returns the highest display_order for stations in a given space.
func (r *StationRepository) GetMaxDisplayOrder(spaceID uuid.UUID) (int, error) {
	var maxOrder int
	err := r.db.Model(&models.Station{}).
		Where("space_id = ?", spaceID).
		Select("COALESCE(MAX(display_order), -1)").
		Scan(&maxOrder).Error
	return maxOrder, err
}
