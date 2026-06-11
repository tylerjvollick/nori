package dtos

import (
	"encoding/json"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNullable_FieldAbsent(t *testing.T) {
	var req UpdateStationRequest
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Mill"}`), &req))
	assert.False(t, req.CostsHour.Set, "absent field should not be marked as set")
}

func TestNullable_FieldNull(t *testing.T) {
	var req UpdateStationRequest
	require.NoError(t, json.Unmarshal([]byte(`{"costsHour":null}`), &req))
	assert.True(t, req.CostsHour.Set)
	assert.False(t, req.CostsHour.Valid)
	assert.Nil(t, req.CostsHour.Ptr())
}

func TestNullable_FieldSetString(t *testing.T) {
	var req UpdateStationRequest
	require.NoError(t, json.Unmarshal([]byte(`{"costsHour":"45.50"}`), &req))
	assert.True(t, req.CostsHour.Set)
	assert.True(t, req.CostsHour.Valid)
	assert.True(t, req.CostsHour.Value.Equal(decimal.RequireFromString("45.50")))
	require.NotNil(t, req.CostsHour.Ptr())
}

func TestNullable_FieldSetNumber(t *testing.T) {
	var req UpdateStationRequest
	require.NoError(t, json.Unmarshal([]byte(`{"costsHour":45.5}`), &req))
	assert.True(t, req.CostsHour.Set)
	assert.True(t, req.CostsHour.Valid)
	assert.True(t, req.CostsHour.Value.Equal(decimal.RequireFromString("45.5")))
}

func TestNullable_InvalidValue(t *testing.T) {
	var req UpdateStationRequest
	err := json.Unmarshal([]byte(`{"costsHour":"not-a-number"}`), &req)
	assert.Error(t, err)
}

func TestNullable_SpaceDefaultLaborRate(t *testing.T) {
	var dto UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"defaultLaborRate":"50.00"}`), &dto))
	assert.True(t, dto.DefaultLaborRate.Set)
	assert.True(t, dto.DefaultLaborRate.Valid)
	assert.True(t, dto.DefaultLaborRate.Value.Equal(decimal.RequireFromString("50")))

	var cleared UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"defaultLaborRate":null}`), &cleared))
	assert.True(t, cleared.DefaultLaborRate.Set)
	assert.False(t, cleared.DefaultLaborRate.Valid)

	var absent UpdateSpaceDTO
	require.NoError(t, json.Unmarshal([]byte(`{"name":"Shop"}`), &absent))
	assert.False(t, absent.DefaultLaborRate.Set)
}
