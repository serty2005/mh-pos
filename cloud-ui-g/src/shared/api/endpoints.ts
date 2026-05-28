import { z } from 'zod';
import { request } from './client';
import {
  restaurantSchema,
  type Restaurant,
} from './schemas';

export function listRestaurants(): Promise<Restaurant[]> {
  return request('/restaurants', z.array(restaurantSchema));
}
