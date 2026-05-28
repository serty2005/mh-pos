import { useCallback, useEffect, useState } from 'react';
import { listEdgeEvents } from '../../shared/api/endpoints';
import type { EdgeEvent } from '../../shared/api/schemas';

export function useEdgeEvents(restaurantId: string) {
  const [events, setEvents] = useState<EdgeEvent[]>([]);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'blocked'>('idle');
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    if (!restaurantId) {
      setEvents([]);
      setStatus('idle');
      return;
    }

    setStatus('loading');
    setError(null);
    try {
      const next = await listEdgeEvents(restaurantId);
      setEvents(next);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  }, [restaurantId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { events, status, error, reload };
}
