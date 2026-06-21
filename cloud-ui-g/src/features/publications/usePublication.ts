import { useCallback, useEffect, useState } from 'react';
import { getPublicationState, listDeliveryStatuses } from '../../shared/api/endpoints';
import type { DeliveryStatus, PublicationSummary } from '../../shared/api/schemas';

export function usePublication(restaurantId: string) {
  const [publication, setPublication] = useState<PublicationSummary | null>(null);
  const [deliveries, setDeliveries] = useState<DeliveryStatus[]>([]);
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
      const [nextPublication, nextDeliveries] = await Promise.all([getPublicationState(restaurantId), listDeliveryStatuses(restaurantId)]);
      setPublication(nextPublication);
      setDeliveries(nextDeliveries);
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  }, [restaurantId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  return { publication, deliveries, status, error, reload };
}
