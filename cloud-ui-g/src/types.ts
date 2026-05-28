import { z } from 'zod';

export const restaurantSchema = z.object({
  id: z.string(),
  name: z.string(),
});

export const restaurantsSchema = z.array(restaurantSchema);

export type Restaurant = z.infer<typeof restaurantSchema>;

export type ProbeStatus = 'loading' | 'ready' | 'blocked';

export type ProbeResult = {
  status: ProbeStatus;
  checkedAt: string;
  route: string;
  restaurantCount: number;
  errorMessageKey?: 'errors.unavailable' | 'errors.invalidResponse';
};
