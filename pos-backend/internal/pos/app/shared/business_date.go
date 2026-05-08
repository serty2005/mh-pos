package shared

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"pos-backend/internal/pos/domain"
)

const (
	DefaultBusinessDayMode              = domain.BusinessDayStandard
	DefaultBusinessDayBoundaryLocalTime = "05:00"
)

// NormalizeBusinessDayConfig приводит ресторанную настройку учетного дня к каноническому виду.
func NormalizeBusinessDayConfig(mode domain.BusinessDayMode, boundaryLocalTime string) (domain.BusinessDayMode, string, error) {
	if mode == "" {
		mode = DefaultBusinessDayMode
	}
	switch mode {
	case domain.BusinessDayStandard, domain.BusinessDay24x7:
	default:
		return "", "", fmt.Errorf("%w: unsupported business_day_mode", domain.ErrInvalid)
	}
	boundaryLocalTime = strings.TrimSpace(boundaryLocalTime)
	if boundaryLocalTime == "" {
		boundaryLocalTime = DefaultBusinessDayBoundaryLocalTime
	}
	if _, err := parseLocalClockHHMM(boundaryLocalTime); err != nil {
		return "", "", err
	}
	return mode, boundaryLocalTime, nil
}

// BusinessDateLocal вычисляет backend-owned учетный день по timezone и режиму ресторана.
func BusinessDateLocal(restaurant domain.Restaurant, instant time.Time) (string, error) {
	mode, boundary, err := NormalizeBusinessDayConfig(restaurant.BusinessDayMode, restaurant.BusinessDayBoundaryLocalTime)
	if err != nil {
		return "", err
	}
	location, err := time.LoadLocation(strings.TrimSpace(restaurant.Timezone))
	if err != nil {
		return "", fmt.Errorf("%w: invalid restaurant timezone", domain.ErrInvalid)
	}
	local := instant.In(location)
	if mode == domain.BusinessDayStandard {
		offset, err := parseLocalClockHHMM(boundary)
		if err != nil {
			return "", err
		}
		local = local.Add(-offset)
	}
	return local.Format(time.DateOnly), nil
}

func parseLocalClockHHMM(value string) (time.Duration, error) {
	hourText, minuteText, ok := strings.Cut(strings.TrimSpace(value), ":")
	if !ok {
		return 0, fmt.Errorf("%w: business_day_boundary_local_time must use HH:MM", domain.ErrInvalid)
	}
	hour, err := strconv.Atoi(hourText)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid business_day_boundary_local_time hour", domain.ErrInvalid)
	}
	minute, err := strconv.Atoi(minuteText)
	if err != nil {
		return 0, fmt.Errorf("%w: invalid business_day_boundary_local_time minute", domain.ErrInvalid)
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return 0, fmt.Errorf("%w: business_day_boundary_local_time is out of range", domain.ErrInvalid)
	}
	return time.Duration(hour)*time.Hour + time.Duration(minute)*time.Minute, nil
}
