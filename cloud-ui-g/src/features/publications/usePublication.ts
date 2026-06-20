import { useCallback, useEffect, useState } from 'react';
import { getPublicationState } from '../../shared/api/endpoints';
import type { PublicationSummary } from '../../shared/api/schemas';

export function usePublication(restaurantId: string) {
  const [publication, setPublication] = useState<PublicationSummary | null>(null);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'blocked'>('idle');
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    if (!restaurantId) {
      setPublication(null);
      setStatus('idle');
      setError(null);
      return;
    }

    setStatus('loading');
    setError(null);
    try {
      const next = await getPublicationState(restaurantId);
      setPublication(next);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  }, [restaurantId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { publication, status, error, reload };
}
