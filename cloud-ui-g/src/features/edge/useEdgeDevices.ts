import { useCallback, useEffect, useState } from 'react';
import {
  assignDeviceToRestaurant,
  generatePairingCode,
  getAssignmentStatus,
  listRestaurantDevices,
  listUnassignedDevices,
} from '../../shared/api/endpoints';
import type { PairingCodeResult, RestaurantEdgeNode, UnassignedEdgeNode } from '../../shared/api/schemas';

export function useEdgeDevices(restaurantId: string) {
  const [devices, setDevices] = useState<UnassignedEdgeNode[]>([]);
  const [restaurantDevices, setRestaurantDevices] = useState<RestaurantEdgeNode[]>([]);
  const [status, setStatus] = useState<'idle' | 'loading' | 'ready' | 'blocked'>('idle');
  const [error, setError] = useState<unknown>(null);
  const [selectedDeviceId, setSelectedDeviceId] = useState('');
  const [assignLoading, setAssignLoading] = useState(false);
  const [pairingLoading, setPairingLoading] = useState(false);
  const [actionError, setActionError] = useState<unknown>(null);
  const [assignmentStatus, setAssignmentStatus] = useState<string>('');
  const [pairingCode, setPairingCode] = useState<PairingCodeResult | null>(null);

  const reload = useCallback(async () => {
    setStatus('loading');
    setError(null);
    try {
      const [unassigned, owned] = await Promise.all([
        listUnassignedDevices(),
        restaurantId ? listRestaurantDevices(restaurantId) : Promise.resolve([]),
      ]);
      setDevices(unassigned);
      setRestaurantDevices(owned);
      setSelectedDeviceId((current) => current || owned[0]?.node_device_id || '');
      setStatus('ready');
    } catch (nextError) {
      setStatus('blocked');
      setError(nextError);
    }
  }, [restaurantId]);

  useEffect(() => {
    void reload();
  }, [reload]);

  const assign = useCallback(async () => {
    if (!restaurantId || !selectedDeviceId) return;
    setAssignLoading(true);
    setActionError(null);
    setPairingCode(null);
    try {
      const result = await assignDeviceToRestaurant(restaurantId, selectedDeviceId);
      setAssignmentStatus(result.status);
      const latest = await getAssignmentStatus(selectedDeviceId);
      setAssignmentStatus(latest.status);
      setSelectedDeviceId(latest.node_device_id);
      await reload();
    } catch (nextError) {
      setActionError(nextError);
    } finally {
      setAssignLoading(false);
    }
  }, [reload, restaurantId, selectedDeviceId]);

  const requestPairingCode = useCallback(async () => {
    if (!restaurantId) return;
    setPairingLoading(true);
    setActionError(null);
    try {
      const result = await generatePairingCode(restaurantId, {});
      setPairingCode(result);
      await reload();
    } catch (nextError) {
      setActionError(nextError);
    } finally {
      setPairingLoading(false);
    }
  }, [reload, restaurantId]);

  return {
    devices,
    restaurantDevices,
    status,
    error,
    selectedDeviceId,
    setSelectedDeviceId,
    reload,
    assign,
    requestPairingCode,
    assignLoading,
    pairingLoading,
    actionError,
    assignmentStatus,
    pairingCode,
    clearPairingCode: () => setPairingCode(null),
  };
}
