package ports

import (
	"context"

	"pos-backend/internal/pos/domain/restaurant"
)

type RestaurantRepository interface {
	CreateRestaurant(context.Context, *restaurant.Restaurant) error
	ListRestaurants(context.Context) ([]restaurant.Restaurant, error)
}
