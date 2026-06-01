import { useCallback, useEffect, useState } from 'react';
import { listRestaurants } from '../../shared/api/endpoints';
import type { Restaurant } from '../../shared/api/schemas';

type RestaurantsStatus = 'idle' | 'loading' | 'ready' | 'blocked';

type UseRestaurantsResult = {
  restaurants: Restaurant[];
  status: RestaurantsStatus;
  error: unknown;
  reload: () => Promise<void>;
};

export function useRestaurants(): UseRestaurantsResult {
  const [restaurants, setRestaurants] = useState<Restaurant[]>([]);
  const [status, setStatus] = useState<RestaurantsStatus>('idle');
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    setStatus('loading');
    setError(null);

    try {
      const data = await listRestaurants();
      setRestaurants(data);
      setStatus('ready');
    } catch (nextError) {
      setRestaurants([]);
      setStatus('blocked');
      setError(nextError);
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { restaurants, status, error, reload };
}
